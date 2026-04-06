package com.pingxin403.cuckoo.flashsale.util;

import java.util.Base64;
import java.util.Map;
import java.util.TreeMap;
import javax.crypto.KeyGenerator;
import javax.crypto.Mac;
import javax.crypto.SecretKey;
import javax.crypto.spec.SecretKeySpec;

public class ApiSignatureUtil {

  private static final String SIGNATURE_ALGORITHM = "HmacSHA256";
  private static final long TIMESTAMP_TOLERANCE_MS = 5 * 60 * 1000;

  public static String generateSignature(
      String secretKey, String method, String path, Map<String, String> params, long timestamp) {

    try {
      Mac mac = Mac.getInstance(SIGNATURE_ALGORITHM);
      SecretKeySpec keySpec = new SecretKeySpec(secretKey.getBytes(), SIGNATURE_ALGORITHM);
      mac.init(keySpec);

      StringBuilder sb = new StringBuilder();
      sb.append(method.toUpperCase()).append("\n");
      sb.append(path).append("\n");
      sb.append(timestamp).append("\n");

      TreeMap<String, String> sortedParams = new TreeMap<>(params);
      for (Map.Entry<String, String> entry : sortedParams.entrySet()) {
        sb.append(entry.getKey()).append("=").append(entry.getValue()).append("&");
      }

      byte[] hash = mac.doFinal(sb.toString().getBytes());
      return Base64.getEncoder().encodeToString(hash);
    } catch (Exception e) {
      throw new RuntimeException("Signature generation failed", e);
    }
  }

  public static boolean verifySignature(
      String secretKey,
      String method,
      String path,
      Map<String, String> params,
      long timestamp,
      String signature) {

    if (Math.abs(System.currentTimeMillis() - timestamp) > TIMESTAMP_TOLERANCE_MS) {
      return false;
    }

    String expectedSignature = generateSignature(secretKey, method, path, params, timestamp);
    return expectedSignature.equals(signature);
  }

  public static long generateTimestamp() {
    return System.currentTimeMillis();
  }

  public static String generateNonce() {
    return java.util.UUID.randomUUID().toString().replace("-", "");
  }
}
