## property

A property is a value of varying type used in Settings. 

A property could be a string, an integer, a boolean. It could be an array, 
it could be a map, it could be a scalar. 

The types and helper methods defined in this package make it easier to work 
with this indeterminate data. 

All data is stored as a JSON-encoded string.

## Data Types

Data in a property can be:
- String
- Integer
- Float
- Boolean

## Value Types

Values of a property can be:
- Scalar
- Array
- Map


## Validation

`validation.go` defines a significant array of validators designed to 
limit the amount of data formatting errors that this package could cause.

## DeepCopy

This package is subject to DeepCopy generation as Properties are stored on 
objects which are stored in k8s. 