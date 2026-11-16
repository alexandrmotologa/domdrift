package browser

import (
	"context"
	"fmt"
)

// StealthScript contains JavaScript injected before any document executes to mask automation.
const StealthScript = `(() => {
	// 1. Mask navigator.webdriver
	Object.defineProperty(navigator, 'webdriver', {
		get: () => undefined,
	});

	// 2. Mock window.chrome object
	if (!window.chrome) {
		window.chrome = {
			app: {
				isInstalled: false,
				InstallState: { DISABLED: 'DISABLED', INSTALLED: 'INSTALLED', NOT_INSTALLED: 'NOT_INSTALLED' },
				RunningState: { CANNOT_RUN: 'CANNOT_RUN', READY_TO_RUN: 'READY_TO_RUN', RUNNING: 'RUNNING' }
			},
			runtime: {
				OnInstalledReason: { CHROME_UPDATE: 'chrome_update', INSTALL: 'install', SHARED_MODULE_UPDATE: 'shared_module_update', UPDATE: 'update' },
				OnRestartRequiredReason: { APP_UPDATE: 'app_update', OS_UPDATE: 'os_update', PERIODIC: 'periodic' },
				PlatformArch: { ARM: 'arm', ARM64: 'arm64', MIPS: 'mips', MIPS64: 'mips64', X86_32: 'x86-32', X86_64: 'x86-64' },
				PlatformNaclArch: { ARM: 'arm', MIPS: 'mips', MIPS64: 'mips64', X86_32: 'x86-32', X86_64: 'x86-64' },
				PlatformOs: { ANDROID: 'android', CROS: 'cros', LINUX: 'linux', MAC: 'mac', OPENBSD: 'openbsd', WIN: 'win' }
			}
		};
	}

	// 3. Mock navigator.plugins and mimeTypes
	Object.defineProperty(navigator, 'plugins', {
		get: () => [
			{ name: 'Chrome PDF Plugin', filename: 'internal-pdf-viewer', description: 'Portable Document Format' },
			{ name: 'Chrome PDF Viewer', filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai', description: '' },
			{ name: 'Native Client', filename: 'internal-nacl-plugin', description: '' }
		]
	});

	// 4. Consistent navigator.languages
	Object.defineProperty(navigator, 'languages', {
		get: () => ['en-US', 'en']
	});

	// 5. Override permissions query
	const originalQuery = window.navigator.permissions ? window.navigator.permissions.query : null;
	if (originalQuery) {
		window.navigator.permissions.query = (parameters) => (
			parameters.name === 'notifications' ?
				Promise.resolve({ state: Notification.permission }) :
				originalQuery(parameters)
		);
	}

	// 6. WebGL Vendor & Renderer spoofing
	const getParameter = WebGLRenderingContext.prototype.getParameter;
	WebGLRenderingContext.prototype.getParameter = function(parameter) {
		if (parameter === 37445) {
			return 'Intel Inc.';
		}
		if (parameter === 37446) {
			return 'Intel(R) Iris(TM) Plus Graphics 640';
		}
		return getParameter.apply(this, arguments);
	};
})();`

// ApplyStealthEvasions registers the evasions script to execute on every new document.
func (s *Session) ApplyStealthEvasions(ctx context.Context) error {
	_, err := s.call(ctx, "Page.addScriptToEvaluateOnNewDocument", map[string]interface{}{
		"source": StealthScript,
	})
	if err != nil {
		return fmt.Errorf("failed registering stealth evasions script: %w", err)
	}
	return nil
}
