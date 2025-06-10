package custom

import (
	"taskd/internal/task"
	"taskd/internal/utils"
)

/**
 * Initialize task pool for KFJob type tasks
 */
func InitK8sExtension(tp *task.TaskPool) error {
	var err error
	tp.Extension, err = utils.InitK8SClient(tp.Config)
	if err != nil {
		return err
	}
	return nil
}
