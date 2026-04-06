package com.pingxin403.cuckoo.flashsale.util;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.util.Base64;
import javax.crypto.Cipher;
import javax.crypto.spec.SecretKeySpec;

public class EncryptionUtil {

  private static final String AES_ALGORITHM = "AES";
  private static final String AES_TRANSFORMATION = "AES/ECB/PKCS5Padding";

  public static String md5(String input) {
    try {
      MessageDigest md = MessageDigest.getInstance("MD5");
      byte[] digest = md.digest(input.getBytes(StandardCharsets.UTF_8));
      StringBuilder sb = new StringBuilder();
      for (byte b : digest) {
        sb.append(String.format("%02x", b));
      }
      return sb.toString();
    } catch (Exception e) {
      throw new RuntimeException("MD5 encryption failed", e);
    }
  }

  public static String sha256(String input) {
    try {
      MessageDigest md = MessageDigest.getInstance("SHA-256");
      byte[] digest = md.digest(input.getBytes(StandardCharsets.UTF_8));
      StringBuilder sb = new StringBuilder();
      for (byte b : digest) {
        sb.append(String.format("%02x", b));
      }
      return sb.toString();
    } catch (Exception e) {
      throw new RuntimeException("SHA-256 encryption failed", e);
    }
  }

  public static String encryptWithAES(String data, String secretKey) {
    try {
      SecretKeySpec keySpec = new SecretKeySpec(secretKey.getBytes(StandardCharsets.UTF_8), AES_ALGORITHM);
      Cipher cipher = Cipher.getInstance(AES_TRANSFORMATION);
      cipher.init(Cipher.ENCRYPT_MODE, keySpec);
      byte[] encrypted = cipher.doFinal(data.getBytes(StandardCharsets.UTF_8));
      return Base64.getEncoder().encodeToString(encrypted);
    } catch (Exception e) {
      throw new RuntimeException("AES encryption failed", e);
    }
  }

  public static String decryptWithAES(String encryptedData, String secretKey) {
    try {
      SecretKeySpec keySpec = new SecretKeySpec(secretKey.getBytes(StandardCharsets.UTF_8), AES_ALGORITHM);
      Cipher cipher = Cipher.getInstance(AES_TRANSFORMATION);
      cipher.init(Cipher.DECRYPT_MODE, keySpec);
      byte[] decrypted = cipher.doFinal(Base64.getDecoder().decode(encryptedData));
      return new String(decrypted, StandardCharsets.UTF_8);
    } catch (Exception e) {
      throw new RuntimeException("AES decryption failed", e);
    }
  }

  public static String maskSensitiveData(String data, int visiblePrefix, int visibleSuffix) {
    if (data == null || data.length() <= visiblePrefix + visibleSuffix) {
      return "***";
    }
    String prefix = data.substring(0, visiblePrefix);
    String suffix = data.substring(data.length() - visibleSuffix);
    int maskLength = data.length() - visiblePrefix - visibleSuffix;
    return prefix + "*".repeat(maskLength) + suffix;
  }
}
