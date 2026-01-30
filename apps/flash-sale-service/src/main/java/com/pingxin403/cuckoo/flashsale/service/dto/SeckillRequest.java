package com.pingxin403.cuckoo.flashsale.service.dto;

/**
 * 秒杀请求记录 Seckill request record for anti-fraud assessment.
 *
 * <p>Contains all information needed to assess the risk of a seckill request, including user
 * identification, device fingerprint, and request metadata.
 *
 * @param userId the user identifier
 * @param skuId the SKU identifier being requested
 * @param quantity the quantity requested
 * @param deviceFingerprint the device fingerprint information
 * @param ipAddress the client IP address
 * @param userAgent the client user agent string
 * @param timestamp the request timestamp in milliseconds
 * @param source the request source (APP, WEB, H5)
 */
public record SeckillRequest(
    String userId,
    String skuId,
    int quantity,
    DeviceFingerprint deviceFingerprint,
    String ipAddress,
    String userAgent,
    long timestamp,
    String source) {

  /**
   * Builder pattern for creating SeckillRequest instances.
   *
   * @return a new Builder instance
   */
  public static Builder builder() {
    return new Builder();
  }

  /** Builder class for SeckillRequest. */
  public static class Builder {
    private String userId;
    private String skuId;
    private int quantity = 1;
    private DeviceFingerprint deviceFingerprint;
    private String ipAddress;
    private String userAgent;
    private long timestamp = System.currentTimeMillis();
    private String source;

    public Builder userId(String userId) {
      this.userId = userId;
      return this;
    }

    public Builder skuId(String skuId) {
      this.skuId = skuId;
      return this;
    }

    public Builder quantity(int quantity) {
      this.quantity = quantity;
      return this;
    }

    public Builder deviceFingerprint(DeviceFingerprint deviceFingerprint) {
      this.deviceFingerprint = deviceFingerprint;
      return this;
    }

    public Builder ipAddress(String ipAddress) {
      this.ipAddress = ipAddress;
      return this;
    }

    public Builder userAgent(String userAgent) {
      this.userAgent = userAgent;
      return this;
    }

    public Builder timestamp(long timestamp) {
      this.timestamp = timestamp;
      return this;
    }

    public Builder source(String source) {
      this.source = source;
      return this;
    }

    public SeckillRequest build() {
      return new SeckillRequest(
          userId, skuId, quantity, deviceFingerprint, ipAddress, userAgent, timestamp, source);
    }
  }

  /**
   * Check if the request has a valid device fingerprint.
   *
   * @return true if device fingerprint is present and has a valid device ID
   */
  public boolean hasValidFingerprint() {
    return deviceFingerprint != null
        && deviceFingerprint.deviceId() != null
        && !deviceFingerprint.deviceId().isBlank();
  }

  /**
   * Get the device ID from the fingerprint, or null if not available.
   *
   * @return the device ID or null
   */
  public String getDeviceId() {
    return deviceFingerprint != null ? deviceFingerprint.deviceId() : null;
  }
}
