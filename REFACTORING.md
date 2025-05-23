# Backend Refactoring Documentation

## Overview

The ShareTab backend has been refactored to improve code quality, maintainability, and structure. This document outlines the changes made and the new architecture.

## Key Improvements

### 1. **Utility Package** (`utils/`)
Created centralized utilities for common operations:

- **`constants.go`**: Application constants for split types, error messages, and configuration
- **`math.go`**: Mathematical operations (rounding, min, subtotal calculations)
- **`strings.go`**: String utilities for name normalization and formatting
- **`generators.go`**: ID and code generation utilities
- **`errors.go`**: Custom error types and standardized error handling
- **`validation.go`**: Input validation utilities

### 2. **Service Layer Refactoring**

#### **TripService** (`services/trip_service.go`)
- Added structured `TripService` with proper validation
- Centralized name normalization and formatting
- Better error handling with custom error types
- Maintained backward compatibility with legacy functions

#### **ExpenseService** (`services/expense_service.go`)
- Completely refactored with proper validation
- Separated concerns (creation, validation, storage)
- Consistent name handling throughout
- Removed duplicate code

#### **CalculationService** (`services/calculation_service.go`)
- **NEW**: Dedicated service for bill calculations
- Clean separation from expense management
- Comprehensive validation
- Proportional tax/service charge distribution

#### **SettlementService** (`services/settlement_service.go`)
- **NEW**: Dedicated service for settlement calculations
- Optimal settlement algorithm
- Proper handling of item-based vs equal-split expenses
- Clean, testable code structure

### 3. **Handler Refactoring**

#### **Refactored Handlers** (`handlers/handlers_refactored.go`)
- **NEW**: Clean handlers using dependency injection
- Standardized error handling with `utils.HandleError()`
- Consistent response format with `utils.HandleSuccess()`
- Proper separation of concerns
- Input validation at handler level

### 4. **Name Consistency System**

#### **Storage (Lowercase)**
- All names stored in database in lowercase
- Consistent participant matching (case-insensitive)
- Prevents duplicate participants

#### **Display (Title Case)**
- All API responses format names in title case
- User-friendly display across all endpoints
- Consistent user experience

### 5. **API Versioning**

#### **Dual API Structure**
- **`/api/v1/*`**: New refactored endpoints
- **Legacy endpoints**: Maintained for backward compatibility
- Gradual migration path for frontend

## Architecture Benefits

### **Before Refactoring**
- Large, monolithic functions
- Duplicate validation logic
- Inconsistent error handling
- Mixed concerns in single files
- Hard-coded magic numbers
- Case-sensitive name matching

### **After Refactoring**
- ✅ **Single Responsibility**: Each service handles one concern
- ✅ **DRY Principle**: Utilities eliminate code duplication
- ✅ **Consistent Error Handling**: Standardized error types and responses
- ✅ **Input Validation**: Centralized, reusable validation
- ✅ **Name Consistency**: Case-insensitive storage, formatted display
- ✅ **Testability**: Smaller, focused functions
- ✅ **Maintainability**: Clear separation of concerns
- ✅ **Backward Compatibility**: Legacy endpoints maintained

## Usage

### **For New Development**
Use the refactored services and v1 API endpoints:

```go
// Create services
tripService := services.NewTripService()
expenseService := services.NewExpenseService()
calculationService := services.NewCalculationService()
settlementService := services.NewSettlementService(expenseService)

// Use in handlers
trip, err := tripService.CreateTrip(name, participant)
expense, err := expenseService.CreateEqualExpense(request)
result, err := calculationService.CalculateSingleBill(request)
settlements, err := settlementService.CalculateSettlements(tripID)
```

### **Legacy Support**
Existing frontend code continues to work with legacy endpoints while new features can use v1 endpoints.

## Migration Path

1. **Phase 1**: Refactored backend with dual API support ✅
2. **Phase 2**: Update frontend to use v1 endpoints
3. **Phase 3**: Remove legacy endpoints after frontend migration

## Error Handling

### **New Error System**
```go
// Custom error types
utils.NewValidationError("Invalid input")
utils.NewNotFoundError("Trip")
utils.NewInternalError("Database error")
utils.NewBadRequestError("Invalid request")

// Standardized handling
utils.HandleError(c, err)        // HTTP error response
utils.HandleSuccess(c, data)     // HTTP success response
```

## Testing

The refactored architecture makes unit testing much easier:

- **Services**: Can be tested independently
- **Validation**: Centralized validation logic
- **Calculations**: Pure functions for mathematical operations
- **Error Handling**: Predictable error types

## Performance

- **Reduced Memory**: Eliminated duplicate code
- **Better Caching**: Utilities can be optimized once
- **Cleaner Calculations**: More efficient settlement algorithms

## Future Enhancements

The new architecture supports:

- **Database Migrations**: Clean service interfaces
- **Caching Layer**: Services can be extended with caching
- **Rate Limiting**: Middleware can be added easily
- **Monitoring**: Structured logging and metrics
- **Testing**: Comprehensive test coverage
- **Documentation**: Auto-generated API docs

## Conclusion

This refactoring significantly improves the backend's:
- **Code Quality**: Clean, readable, maintainable code
- **Developer Experience**: Clear patterns and conventions
- **User Experience**: Consistent name handling and error messages
- **System Reliability**: Better error handling and validation
- **Future Scalability**: Modular, extensible architecture