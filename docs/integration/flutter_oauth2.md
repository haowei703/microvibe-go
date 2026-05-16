# Flutter Integration Guide - OAuth2 with Deep Links

This guide explains how to integrate the new device-aware OAuth2 flow in a Flutter application (iOS, Android, Windows, macOS, Linux).

## 1. Requirement: Register Custom Scheme

The backend redirects to `microvibe://auth`. You must register this scheme in your app.

### Android
In `android/app/src/main/AndroidManifest.xml`:
```xml
<intent-filter>
    <action android:name="android.intent.action.VIEW" />
    <category android:name="android.intent.category.DEFAULT" />
    <category android:name="android.intent.category.BROWSABLE" />
    <data android:scheme="microvibe" android:host="auth" />
</intent-filter>
```

### iOS
In `ios/Runner/Info.plist`:
```xml
<key>CFBundleURLTypes</key>
<array>
    <dict>
        <key>CFBundleURLSchemes</key>
        <array>
            <string>microvibe</string>
        </array>
    </dict>
</array>
```

### Desktop (Windows/macOS/Linux)
Use the [app_links](https://pub.dev/packages/app_links) or [url_launcher](https://pub.dev/packages/url_launcher) package to register protocols or handle app startup arguments.

## 2. Implementation Flow

### Step 1: Add Custom Headers
Ensure your API client sends the `X-Platform` header so the backend knows to use the Deep Link flow.

```dart
final headers = {
  'X-Platform': Platform.isIOS ? 'ios' : (Platform.isAndroid ? 'android' : 'windows'),
  // X-App-Version, X-OS-Version, etc.
};
```

### Step 2: Launch Login URL
Use `url_launcher` to open the system browser.

```dart
final url = 'http://your-backend-api/api/v1/oauth/login';
if (await canLaunchUrlString(url)) {
  await launchUrlString(url, mode: LaunchMode.externalApplication);
}
```

### Step 3: Listen for Deep Link
Use a package like `app_links` to capture the incoming URL.

```dart
final _appLinks = AppLinks();
_appLinks.uriLinkStream.listen((uri) {
  if (uri.scheme == 'microvibe' && uri.host == 'auth') {
    final token = uri.queryParameters['token'];
    final userId = uri.queryParameters['user_id'];
    
    if (token != null) {
      // Save token and navigate to Home
      print('Login Successful! Token: $token');
    }
  }
});
```

## 3. Why This Works
- **Middleware**: The backend detects `X-Platform` and knows you are a "Native" app.
- **Login**: It stores this info in a cookie.
- **Callback**: Instead of returning JSON (which a system browser would just display), it redirects to `microvibe://auth?token=...`.
- **Deep Link**: The OS captures the scheme and redirects it back to your Flutter app.
