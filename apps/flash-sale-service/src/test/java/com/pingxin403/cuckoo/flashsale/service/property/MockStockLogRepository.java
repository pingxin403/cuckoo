package com.pingxin403.cuckoo.flashsale.service.property;

import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.List;
import java.util.Optional;

import org.springframework.data.domain.Example;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.data.domain.Sort;
import org.springframework.data.repository.query.FluentQuery;

import com.pingxin403.cuckoo.flashsale.model.StockLog;
import com.pingxin403.cuckoo.flashsale.model.enums.StockOperation;
import com.pingxin403.cuckoo.flashsale.repository.StockLogRepository;

/** Mock implementation of StockLogRepository for testing. */
public class MockStockLogRepository implements StockLogRepository {

  private final List<StockLog> logs = new ArrayList<>();

  @Override
  public StockLog save(StockLog entity) {
    logs.add(entity);
    return entity;
  }

  @Override
  public <S extends StockLog> List<S> saveAll(Iterable<S> entities) {
    entities.forEach(logs::add);
    return (List<S>) logs;
  }

  @Override
  public Optional<StockLog> findById(Long aLong) {
    return Optional.empty();
  }

  @Override
  public boolean existsById(Long aLong) {
    return false;
  }

  @Override
  public List<StockLog> findAll() {
    return logs;
  }

  @Override
  public List<StockLog> findAllById(Iterable<Long> longs) {
    return List.of();
  }

  @Override
  public long count() {
    return logs.size();
  }

  @Override
  public void deleteById(Long aLong) {}

  @Override
  public void delete(StockLog entity) {}

  @Override
  public void deleteAllById(Iterable<? extends Long> longs) {}

  @Override
  public void deleteAll(Iterable<? extends StockLog> entities) {}

  @Override
  public void deleteAll() {
    logs.clear();
  }

  @Override
  public void flush() {}

  @Override
  public <S extends StockLog> S saveAndFlush(S entity) {
    logs.add(entity);
    return entity;
  }

  @Override
  public <S extends StockLog> List<S> saveAllAndFlush(Iterable<S> entities) {
    return saveAll(entities);
  }

  @Override
  public void deleteAllInBatch(Iterable<StockLog> entities) {}

  @Override
  public void deleteAllByIdInBatch(Iterable<Long> longs) {}

  @Override
  public void deleteAllInBatch() {}

  @Override
  public StockLog getOne(Long aLong) {
    return null;
  }

  @Override
  public StockLog getById(Long aLong) {
    return null;
  }

  @Override
  public StockLog getReferenceById(Long aLong) {
    return null;
  }

  @Override
  public <S extends StockLog> Optional<S> findOne(Example<S> example) {
    return Optional.empty();
  }

  @Override
  public <S extends StockLog> List<S> findAll(Example<S> example) {
    return List.of();
  }

  @Override
  public <S extends StockLog> List<S> findAll(Example<S> example, Sort sort) {
    return List.of();
  }

  @Override
  public <S extends StockLog> Page<S> findAll(Example<S> example, Pageable pageable) {
    return Page.empty();
  }

  @Override
  public <S extends StockLog> long count(Example<S> example) {
    return 0;
  }

  @Override
  public <S extends StockLog> boolean exists(Example<S> example) {
    return false;
  }

  @Override
  public <S extends StockLog, R> R findBy(
      Example<S> example,
      java.util.function.Function<FluentQuery.FetchableFluentQuery<S>, R> queryFunction) {
    return null;
  }

  @Override
  public List<StockLog> findAll(Sort sort) {
    return logs;
  }

  @Override
  public Page<StockLog> findAll(Pageable pageable) {
    return Page.empty();
  }

  @Override
  public List<StockLog> findBySkuId(String skuId) {
    return logs.stream().filter(log -> log.getSkuId().equals(skuId)).toList();
  }

  @Override
  public List<StockLog> findByOrderId(String orderId) {
    return logs.stream().filter(log -> log.getOrderId().equals(orderId)).toList();
  }

  @Override
  public List<StockLog> findBySkuIdAndOperation(String skuId, StockOperation operation) {
    return logs.stream()
        .filter(log -> log.getSkuId().equals(skuId) && log.getOperation().equals(operation))
        .toList();
  }

  @Override
  public Optional<StockLog> findFirstBySkuIdOrderByCreatedAtDesc(String skuId) {
    return logs.stream()
        .filter(log -> log.getSkuId().equals(skuId))
        .max((a, b) -> a.getCreatedAt().compareTo(b.getCreatedAt()));
  }

  @Override
  public Optional<StockLog> findFirstByOrderIdOrderByCreatedAtDesc(String orderId) {
    return logs.stream()
        .filter(log -> log.getOrderId().equals(orderId))
        .max((a, b) -> a.getCreatedAt().compareTo(b.getCreatedAt()));
  }

  @Override
  public List<StockLog> findBySkuIdAndTimeRange(
      String skuId, LocalDateTime startTime, LocalDateTime endTime) {
    return logs.stream()
        .filter(
            log ->
                log.getSkuId().equals(skuId)
                    && !log.getCreatedAt().isBefore(startTime)
                    && !log.getCreatedAt().isAfter(endTime))
        .toList();
  }

  @Override
  public long countDeductionsBySkuId(String skuId) {
    return logs.stream()
        .filter(log -> log.getSkuId().equals(skuId) && log.getOperation() == StockOperation.DEDUCT)
        .count();
  }

  @Override
  public long countRollbacksBySkuId(String skuId) {
    return logs.stream()
        .filter(
            log -> log.getSkuId().equals(skuId) && log.getOperation() == StockOperation.ROLLBACK)
        .count();
  }

  @Override
  public long sumDeductedQuantityBySkuId(String skuId) {
    return logs.stream()
        .filter(log -> log.getSkuId().equals(skuId) && log.getOperation() == StockOperation.DEDUCT)
        .mapToLong(StockLog::getQuantity)
        .sum();
  }

  @Override
  public long sumRolledBackQuantityBySkuId(String skuId) {
    return logs.stream()
        .filter(
            log -> log.getSkuId().equals(skuId) && log.getOperation() == StockOperation.ROLLBACK)
        .mapToLong(StockLog::getQuantity)
        .sum();
  }

  @Override
  public boolean existsDeductionByOrderId(String orderId) {
    return logs.stream()
        .anyMatch(
            log -> log.getOrderId().equals(orderId) && log.getOperation() == StockOperation.DEDUCT);
  }

  @Override
  public boolean existsRollbackByOrderId(String orderId) {
    return logs.stream()
        .anyMatch(
            log ->
                log.getOrderId().equals(orderId) && log.getOperation() == StockOperation.ROLLBACK);
  }
}
