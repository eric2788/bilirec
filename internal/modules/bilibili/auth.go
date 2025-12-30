package bilibili

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	bili "github.com/CuteReimu/bilibili/v2"
	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/go-resty/resty/v2"
)

func (c *Client) loadCookiesOrLogin(ctx context.Context, cfg *config.Config) error {
	if cfg.AnonymousLogin {
		logger.Info("using anonymous login, skipping bilibili login process")
		return nil
	}

	defer c.syncCookies()

	if cookie, token, err := c.loadOfflineCredentials(); err == nil {
		c.SetCookiesString(cookie)
		c.wbi.WithCookies(c.GetCookies())
		c.refreshToken = token

		if acc, err := c.GetAccountInformation(); err == nil {
			logger.Infof("loaded cookies for user: %s (mid: %d)", acc.Uname, acc.Mid)
			return c.refreshCookiesIfRequired()
		} else {
			logger.Warnf("failed to get account information with loaded cookies: %v", err)
		}

	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error reading cookie file: %v", err)
	}

	logger.Info("starting bilibili login process")

	qrcode, err := c.GetQRCode()
	if err != nil {
		return fmt.Errorf("error getting qrcode: %v", err)
	}

	logger.Info("please scan the qrcode to login:")
	qrcode.Print()

	// blocking thread
	result, err := c.LoginWithQRCode(bili.LoginWithQRCodeParam{
		QrcodeKey: qrcode.QrcodeKey,
	})

	if err != nil {
		return fmt.Errorf("error logging in with qrcode: %v", err)
	} else if result.Code != 0 {
		return fmt.Errorf("login failed: %s (code %d)", result.Message, result.Code)
	}

	if acc, err := c.GetAccountInformation(); err != nil {
		return fmt.Errorf("error getting account information after login: %v, please try again.", err)
	} else {
		logger.Infof("login successful. logged in as %s (mid: %d)", acc.Uname, acc.Mid)
	}

	if err := c.writeRefreshTokenToFile(result.RefreshToken); err != nil {
		return err
	}
	go c.refreshCookiesPeriodically(ctx, 10*time.Minute)
	return c.writerCookiesToFile()
}

func (c *Client) refreshCookiesIfRequired() error {
	info, err := c.GetWebCookieRefreshInfo()
	if err != nil {
		return fmt.Errorf("error getting cookie refresh info: %v", err)
	}
	if !info.Refresh {
		logger.Info("cookies do not require refresh")
		return nil
	}
	csrfResult, err := c.GetWebCookieRefreshCsrf(bili.GetWebCookieRefreshCsrfParam{
		Timestamp: info.Timestamp,
	})
	if err != nil {
		return fmt.Errorf("error getting cookie refresh csrf: %v", err)
	}
	refreshed, err := c.RefreshCookie(bili.RefreshCookieParam{
		RefreshToken: c.refreshToken,
		RefreshCsrf:  csrfResult.RefreshCsrf,
	})
	if err != nil {
		return fmt.Errorf("error refreshing cookies: %v", err)
	} else {
		logger.Info("cookies refreshed successfully")
		c.syncCookies()
	}

	if err := c.writeRefreshTokenToFile(refreshed.RefreshToken); err != nil {
		return err
	}
	return c.writerCookiesToFile()
}

func (c *Client) writerCookiesToFile() error {
	cookieStr := c.GetCookiesString()
	if err := os.WriteFile(c.cookiePath, []byte(cookieStr), 0600); err != nil {
		return fmt.Errorf("error writing cookie file: %v", err)
	}
	return nil
}

func (c *Client) writeRefreshTokenToFile(refreshToken string) error {
	if err := os.WriteFile(c.refreshToken, []byte(refreshToken), 0600); err != nil {
		return fmt.Errorf("error writing refresh token file: %v", err)
	}
	return nil
}

func (c *Client) loadOfflineCredentials() (cookie string, refreshToken string, err error) {

	cookieBytes, err := os.ReadFile(c.cookiePath)
	if os.IsNotExist(err) {
		cookie = ""
	} else if err != nil {
		err = fmt.Errorf("error reading cookie file: %v", err)
	} else {
		cookie = string(cookieBytes)
	}

	refreshTokenBytes, err := os.ReadFile(c.refreshTokenPath)
	if os.IsNotExist(err) {
		refreshToken = ""
	} else if err != nil {
		err = fmt.Errorf("error reading refresh token file: %v", err)
	} else {
		refreshToken = string(refreshTokenBytes)
	}

	return
}

func (c *Client) refreshCookiesPeriodically(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := c.refreshCookiesIfRequired(); err != nil {
				logger.Error(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *Client) syncCookies() {
	mainCookies := c.GetCookies()
	// 更新或添加 cookies 到 liveClient
	go syncCookieToClient(c.liveClient, mainCookies)
	// 同樣處理 liveStreamClient
	go syncCookieToClient(c.liveStreamClient, mainCookies)
}

func syncCookieToClient(client *resty.Client, cookies []*http.Cookie) {
	for _, mainCookie := range cookies {
		found := false
		for j, streamCookie := range client.Cookies {
			if mainCookie.Name == streamCookie.Name {
				client.Cookies[j] = mainCookie
				found = true
				break
			}
		}
		if !found {
			client.Cookies = append(client.Cookies, mainCookie)
		}
	}
}
