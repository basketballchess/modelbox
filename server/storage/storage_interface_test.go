package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	MODEL_NAME = "gpt3"
	OWNER      = "unicorn@modelbox.io"
	TASK       = "translate"
	NAMESPACE  = "ai/langtech/translation"
)

type StorageInterfaceTestSuite struct {
	t *testing.T

	storageIf MetadataStorage
}

func (s *StorageInterfaceTestSuite) TestCreateExperiment() {
	since := time.Now()
	ctx := context.Background()
	e := NewExperiment(MODEL_NAME, OWNER, NAMESPACE, "xyz", Pytorch)
	_, err := s.storageIf.CreateExperiment(context.Background(), e)
	assert.Nil(s.t, err)
	experiments, err := s.storageIf.ListExperiments(context.Background(), e.Namespace)
	assert.Nil(s.t, err)
	assert.Equal(s.t, 1, len(experiments))
	assert.Equal(s.t, MODEL_NAME, experiments[0].Name)
	assert.Equal(s.t, OWNER, experiments[0].Owner)
	assert.Equal(s.t, NAMESPACE, experiments[0].Namespace)
	assert.Equal(s.t, "xyz", experiments[0].ExternalId)

	// Check for mutation events
	changes, err := s.storageIf.ListChanges(ctx, NAMESPACE, since)
	assert.Nil(s.t, err)
	assert.Equal(s.t, len(changes), 1)
}

func (s *StorageInterfaceTestSuite) TestCreateCheckpoint() {
	metrics := SerializableMetrics(map[string]float32{"val_loss": 0.041, "train_accu": 98.01})
	e := NewExperiment("quartznet-lid", "owner@email", "langtech", "xyz", Pytorch)
	c := NewCheckpoint(e.Id, 45, metrics)
	chk, err := s.storageIf.CreateCheckpoint(context.Background(), c)
	assert.Nil(s.t, err)
	assert.Equal(s.t, c.Id, chk.CheckpointId)
	checkpoints, err := s.storageIf.ListCheckpoints(context.Background(), e.Id)
	assert.Nil(s.t, err)
	assert.Equal(s.t, 1, len(checkpoints))
	assert.Equal(s.t, chk.CheckpointId, checkpoints[0].Id)
	assert.Equal(s.t, e.Id, checkpoints[0].ExperimentId)
	assert.Equal(s.t, uint64(45), checkpoints[0].Epoch)
	assert.Equal(s.t, metrics, checkpoints[0].Metrics)
}

func (s *StorageInterfaceTestSuite) TestObjectCreateIdempotency() {
	ctx := context.Background()
	metrics := SerializableMetrics(map[string]float32{"val_loss": 0.041, "train_accu": 98.01})
	e := NewExperiment("quartznet-lid", "owner@email", "langtech", "xyz", Pytorch)
	result1, err := s.storageIf.CreateExperiment(ctx, e)
	assert.Nil(s.t, err)
	result2, err := s.storageIf.CreateExperiment(ctx, e)
	assert.Nil(s.t, err)
	assert.Equal(s.t, result1.ExperimentId, result2.ExperimentId)

	c := NewCheckpoint(e.Id, 45, metrics)
	chk1, err := s.storageIf.CreateCheckpoint(ctx, c)
	assert.Nil(s.t, err)
	chk2, err := s.storageIf.CreateCheckpoint(ctx, c)
	assert.Nil(s.t, err)
	assert.Equal(s.t, chk1.CheckpointId, chk2.CheckpointId)

	m1 := NewModel(MODEL_NAME, e.Owner, NAMESPACE, TASK, "description")
	resp1, err := s.storageIf.CreateModel(ctx, m1)
	assert.Nil(s.t, err)
	m2 := NewModel(MODEL_NAME, e.Owner, NAMESPACE, TASK, "description")
	resp2, err := s.storageIf.CreateModel(ctx, m2)
	assert.Nil(s.t, err)
	assert.Equal(s.t, resp1.ModelId, resp2.ModelId)
}

func (s *StorageInterfaceTestSuite) TestCreateModel() {
	description := "a large translation model based on gpt3"
	m := NewModel("blender", OWNER, NAMESPACE, TASK, description)
	blob1 := NewFileMetadata(m.Id, "/foo/bar", "checksum123", TextFile, 0, 0)
	blob2 := NewFileMetadata(m.Id, "/foo/pipe", "checksum345", ModelFile, 0, 0)
	m.SetFiles([]*FileMetadata{blob1, blob2})
	ctx := context.Background()
	_, err := s.storageIf.CreateModel(ctx, m)
	assert.Nil(s.t, err)

	m1, err := s.storageIf.GetModel(ctx, m.Id)
	assert.Nil(s.t, err)
	assert.Equal(s.t, description, m1.Description)
	assert.Equal(s.t, NAMESPACE, m1.Namespace)
}

func (s *StorageInterfaceTestSuite) TestListModels() {
	description := "a large translation model based on gpt3"
	namespace := "namespace-x"
	m := NewModel("blender", OWNER, namespace, TASK, description)
	blob1 := NewFileMetadata(m.Id, "/foo/bar", "checksum123", TextFile, 0, 0)
	blob2 := NewFileMetadata(m.Id, "/foo/pipe", "checksum345", ModelFile, 0, 0)
	m.SetFiles([]*FileMetadata{blob1, blob2})
	ctx := context.Background()
	_, err := s.storageIf.CreateModel(ctx, m)
	assert.Nil(s.t, err)

	models, err := s.storageIf.ListModels(ctx, namespace)
	assert.Nil(s.t, err)
	assert.Equal(s.t, 1, len(models))
	assert.Equal(s.t, 2, len(models[0].Files))
}

func (s *StorageInterfaceTestSuite) TestCreateModelVersion() {
	blobs := []*FileMetadata{}
	mvName := "test-version"
	version := "1"
	description := "testing"
	uniqueTags := SerializableTags([]string{"foo", "bar"})
	mv := NewModelVersion(
		mvName,
		MODEL_NAME,
		version,
		description,
		Pytorch,
		blobs,
		uniqueTags,
	)
	_, err := s.storageIf.CreateModelVersion(context.Background(), mv)
	assert.Nil(s.t, err)

	mv1, err := s.storageIf.GetModelVersion(context.Background(), mv.Id)
	assert.Nil(s.t, err)
	assert.Equal(s.t, mvName, mv1.Name)
	assert.Equal(s.t, version, mv1.Version)
	assert.Equal(s.t, description, mv1.Description)
	assert.Equal(s.t, uniqueTags, mv1.UniqueTags)
}

func (s *StorageInterfaceTestSuite) TestWriteBlobs() {
	ctx := context.Background()
	blob1 := NewFileMetadata(MODEL_NAME, "/foo/bar", "checksum123", TextFile, 0, 0)
	blob2 := NewFileMetadata(MODEL_NAME, "/foo/pipe", "checksum345", ModelFile, 0, 0)
	blobs := []*FileMetadata{blob1, blob2}
	err := s.storageIf.WriteFiles(ctx, blobs)
	assert.Nil(s.t, err)

	// Test Get Blobs for ParentID
	blobsOut, err := s.storageIf.GetFiles(ctx, MODEL_NAME)
	assert.Nil(s.t, err)
	assert.Equal(s.t, 2, len(blobsOut))
	assert.Equal(s.t, "/foo/bar", blobsOut[0].Path)
	assert.Equal(s.t, "/foo/pipe", blobsOut[1].Path)

	// Test Get Blob with ID
	blob3, err := s.storageIf.GetFile(ctx, blob1.Id)
	assert.Nil(s.t, err)
	assert.Equal(s.t, blob1.Id, blob3.Id)
}

func (s *StorageInterfaceTestSuite) TestUpdateMetadata() {
	ctx := context.Background()

	// Write Metadata
	val1, _ := structpb.NewValue(1)
	meta1 := map[string]*structpb.Value{"/tmp/foo": val1}
	err := s.storageIf.UpdateMetadata(ctx, "parent-id1", meta1)
	assert.Nil(s.t, err)

	val2, _ := structpb.NewValue(map[string]interface{}{"name1": "val1", "name2": 5})
	complexVal := map[string]*structpb.Value{"/tmp/hola": val2}
	err = s.storageIf.UpdateMetadata(ctx, "parent-id2", complexVal)
	assert.Nil(s.t, err)

	// Get Metadata
	meta3, err := s.storageIf.ListMetadata(ctx, "parent-id1")
	assert.Nil(s.t, err)
	assert.Equal(s.t, 1, len(meta3))

	meta4, err := s.storageIf.ListMetadata(ctx, "parent-id2")
	assert.Nil(s.t, err)
	assert.Equal(s.t, 1, len(meta4))
}
