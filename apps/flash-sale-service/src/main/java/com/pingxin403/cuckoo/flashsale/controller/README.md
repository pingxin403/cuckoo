# SeckillController API Documentation

## Overview

The SeckillController provides REST endpoints for flash sale (seckill) operations, implementing the complete three-layer funnel model:

1. **Anti-fraud layer** - Risk assessment and rate limiting
2. **Queue layer** - Token bucket based traffic control
3. **Inventory layer** - Atomic stock deduction
4. **Async processing** - Kafka message queue for order persistence

## Endpoints

### Seckill Entry

**POST** `/api/seckill/{skuId}`

Initiates a flash sale purchase request.

**Parameters:**
- `skuId` (path) - SKU identifier
- `userId` (query, required) - User identifier
- `quantity` (query, optional, default=1) - Quantity to purchase
- `deviceId` (query, optional) - Device identifier for anti-fraud
- `captchaCode` (query, optional) - Captcha code if required
- `source` (query, optional, default=WEB) - Request source (WEB, APP, H5)

**Response Codes:**
- `200` - Success, order created
- `202` - Queuing, retry later
- `403` - Blocked by anti-fraud
- `410` - Sold out
- `422` - Purchase limit exceeded
- `423` - Captcha required
- `503` - System busy

**Example Request:**
```bash
POST /api/seckill/sku123?userId=user456&quantity=1&source=WEB&deviceId=device789
```

**Example Success Response:**
```json
{
  "code": 200,
  "message": "秒杀成功",
  "orderId": "order-1234567890",
  "remainingStock": 99,
  "estimatedWait": null,
  "queueToken": null
}
```

**Example Queuing Response:**
```json
{
  "code": 202,
  "message": "排队中",
  "orderId": null,
  "remainingStock": null,
  "estimatedWait": 5,
  "queueToken": "queue-1234567890"
}
```

### Order Status Query

**GET** `/api/seckill/status/{orderId}`

Queries the status of an order or queue position.

**Parameters:**
- `orderId` (path) - Order identifier or queue token

**Example Request:**
```bash
GET /api/seckill/status/order-1234567890
```

**Example Response:**
```json
{
  "orderId": "order-1234567890",
  "status": "PENDING_PAYMENT",
  "message": "待支付"
}
```

## Activity Management Endpoints

### Create Activity

**POST** `/api/seckill/activity`

Creates a new flash sale activity.

**Request Body:**
```json
{
  "skuId": "sku123",
  "activityName": "Flash Sale Event",
  "totalStock": 100,
  "startTime": "2024-01-01T10:00:00",
  "endTime": "2024-01-01T12:00:00",
  "purchaseLimit": 2
}
```

**Response:** `201 Created` with activity details

### Get Activity

**GET** `/api/seckill/activity/{activityId}`

Retrieves activity details by ID.

**Response:** `200 OK` with activity details or `404 Not Found`

### Get All Activities

**GET** `/api/seckill/activity`

Retrieves all activities.

**Response:** `200 OK` with array of activities

### Update Activity

**PUT** `/api/seckill/activity/{activityId}`

Updates an existing activity.

**Request Body:**
```json
{
  "activityName": "Updated Activity Name",
  "totalStock": 150,
  "startTime": "2024-01-01T10:00:00",
  "endTime": "2024-01-01T14:00:00",
  "purchaseLimit": 3
}
```

**Response:** `200 OK` with updated activity or `404 Not Found`

### Delete Activity

**DELETE** `/api/seckill/activity/{activityId}`

Deletes an activity.

**Response:** `204 No Content` or `404 Not Found`

### Start Activity

**POST** `/api/seckill/activity/{activityId}/start`

Manually starts an activity.

**Response:**
```json
{
  "success": true,
  "message": "活动已开启"
}
```

### End Activity

**POST** `/api/seckill/activity/{activityId}/end`

Manually ends an activity.

**Response:**
```json
{
  "success": true,
  "message": "活动已结束"
}
```

## Error Handling

All endpoints follow consistent error response format:

```json
{
  "code": 422,
  "message": "超过限购数量",
  "orderId": null,
  "remainingStock": null,
  "estimatedWait": null,
  "queueToken": null
}
```

## Integration

The controller integrates the following services:

- **InventoryService** - Redis-based atomic stock management
- **QueueService** - Token bucket rate limiting and queue management
- **OrderService** - Order lifecycle management
- **AntiFraudService** - Multi-layer risk assessment
- **ActivityService** - Activity configuration and lifecycle
- **OrderMessageProducer** - Kafka message production for async processing

## Testing

Unit tests: `SeckillControllerTest`
Integration tests: `SeckillControllerIntegrationTest` (requires Docker for Testcontainers)

Run tests:
```bash
./gradlew test --tests SeckillControllerTest
```

## Performance Considerations

- The controller is designed for high concurrency (100K+ QPS)
- All operations are non-blocking where possible
- Redis operations use Lua scripts for atomicity
- Kafka is used for async order persistence to reduce latency
- Anti-fraud checks are optimized for minimal overhead

## Security

- IP address extraction supports X-Forwarded-For and X-Real-IP headers
- Device fingerprinting for fraud detection
- Captcha verification for suspicious users
- Rate limiting at multiple layers (L1 gateway, L2 application, L3 risk control)
