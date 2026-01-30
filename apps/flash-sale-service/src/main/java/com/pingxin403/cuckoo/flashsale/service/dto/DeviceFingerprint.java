package com.pingxin403.cuckoo.flashsale.service.dto;

/**
 * 设备指纹记录 Device fingerprint record for anti-fraud detection.
 *
 * <p>Contains device identification information used to detect suspicious behavior patterns and
 * identify potential fraud attempts.
 *
 * <p>Redis Key Pattern: device_risk:{deviceId} -> Hash { score: Integer, lastSeen: Long,
 * requestCount: Integer } -> TTL: 24小时
 *
 * <p>Validates Requirements: 3.3
 *
 * @param deviceId unique device identifier (generated from device characteristics)
 * @param platform the platform type (iOS, Android, Web, etc.)
 * @param browserName the browser name (Chrome, Safari, Firefox, etc.)
 * @param browserVersion the browser version
 * @param osName the operating system name
 * @param osVersion the operating system version
 * @param screenResolution the screen resolution (e.g., "1920x1080")
 * @param timezone the timezone offset in minutes
 * @param language the browser language setting
 * @param canvasFingerprint the canvas fingerprint hash
 * @param webglFingerprint the WebGL fingerprint hash
 */
public record DeviceFingerprint(
    String deviceId,
    String platform,
    String browserName,
    String browserVersion,
    String osName,
    String osVersion,
    String screenResolution,
    Integer timezone,
    String language,
    String canvasFingerprint,
    String webglFingerprint) {

  /**
   * Builder pattern for creating DeviceFingerprint instances.
   *
   * @return a new Builder instance
   */
  public static Builder builder() {
    return new Builder();
  }

  /** Builder class for DeviceFingerprint. */
  public static class Builder {
    private String deviceId;
    private String platform;
    private String browserName;
    private String browserVersion;
    private String osName;
    private String osVersion;
    private String screenResolution;
    private Integer timezone;
    private String language;
    private String canvasFingerprint;
    private String webglFingerprint;

    public Builder deviceId(String deviceId) {
      this.deviceId = deviceId;
      return this;
    }

    public Builder platform(String platform) {
      this.platform = platform;
      return this;
    }

    public Builder browserName(String browserName) {
      this.browserName = browserName;
      return this;
    }

    public Builder browserVersion(String browserVersion) {
      this.browserVersion = browserVersion;
      return this;
    }

    public Builder osName(String osName) {
      this.osName = osName;
      return this;
    }

    public Builder osVersion(String osVersion) {
      this.osVersion = osVersion;
      return this;
    }

    public Builder screenResolution(String screenResolution) {
      this.screenResolution = screenResolution;
      return this;
    }

    public Builder timezone(Integer timezone) {
      this.timezone = timezone;
      return this;
    }

    public Builder language(String language) {
      this.language = language;
      return this;
    }

    public Builder canvasFingerprint(String canvasFingerprint) {
      this.canvasFingerprint = canvasFingerprint;
      return this;
    }

    public Builder webglFingerprint(String webglFingerprint) {
      this.webglFingerprint = webglFingerprint;
      return this;
    }

    public DeviceFingerprint build() {
      return new DeviceFingerprint(
          deviceId,
          platform,
          browserName,
          browserVersion,
          osName,
          osVersion,
          screenResolution,
          timezone,
          language,
          canvasFingerprint,
          webglFingerprint);
    }
  }

  /**
   * Check if this fingerprint has a valid device ID.
   *
   * @return true if device ID is present and not blank
   */
  public boolean hasValidDeviceId() {
    return deviceId != null && !deviceId.isBlank();
  }

  /**
   * Create a simple fingerprint with only device ID.
   *
   * @param deviceId the device identifier
   * @return a DeviceFingerprint with only deviceId set
   */
  public static DeviceFingerprint ofDeviceId(String deviceId) {
    return builder().deviceId(deviceId).build();
  }
}
