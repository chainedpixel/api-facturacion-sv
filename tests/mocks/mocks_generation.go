package mocks

//go:generate mockgen -destination=./auth_manager_mock.go -package=mocks github.com/chainedpixel/ordo-factus/internal/domain/auth AuthManager
//go:generate mockgen -destination=./dte_service_mock.go -package=mocks github.com/chainedpixel/ordo-factus/internal/domain/ports DTEService
//go:generate mockgen -destination=./transmitter_mock.go -package=mocks github.com/chainedpixel/ordo-factus/internal/application/ports BaseTransmitter
//go:generate mockgen -destination=./seq_number_mock.go -package=mocks github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents SequentialNumberManager
//go:generate mockgen -destination=./dte_creator_mock.go -package=mocks github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents DTEManager
//go:generate mockgen -destination=./dte_repository_mock.go -package=mocks github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents DTERepositoryPort
//go:generate mockgen -destination=./seqnumber_repository_mock.go -package=mocks github.com/chainedpixel/ordo-factus/internal/domain/ports SequentialNumberRepositoryPort
//go:generate mockgen -destination=./reserved_sequence_repository_mock.go -package=mocks github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents ReservedSequenceRepositoryPort
//go:generate mockgen -destination=./contingency_manager_mock.go -package=mocks github.com/chainedpixel/ordo-factus/internal/domain/dte/contingency ContingencyManager
//go:generate mockgen -destination=./auth_repository_mock.go -package=mocks github.com/chainedpixel/ordo-factus/internal/domain/auth AuthRepositoryPort
//go:generate mockgen -destination=./contingency_repository_mock.go -package=mocks github.com/chainedpixel/ordo-factus/internal/domain/dte/contingency ContingencyRepositoryPort
//go:generate mockgen -destination=./contingency_event_sender_mock.go -package=mocks github.com/chainedpixel/ordo-factus/internal/domain/dte/contingency ContingencyEventSender
//go:generate mockgen -destination=./cache_manager_mock.go -package=mocks github.com/chainedpixel/ordo-factus/internal/domain/ports CacheManager
