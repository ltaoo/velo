package desktopapp

import "github.com/ltaoo/velo"

func registerTaskRoutes(b *velo.Box) {
	b.Get("/api/tasks", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		index, err := loadTaskIndex(ctx)
		if err != nil {
			return c.Error(err.Error())
		}
		entries := taskIndexEntries(index)
		for i, entry := range entries {
			if isPrivateAndLocked(ctx, entry.Private) {
				entries[i].Title = "[PRIVATE]"
				entries[i].Tags = []string{}
				entries[i].Contexts = []string{}
			}
		}
		return c.Ok(velo.H{
			"index": index,
			"tasks": entries,
		})
	})

	b.Get("/api/tasks/get", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		task, err := getVaultTask(ctx, c.Query("id"))
		if err != nil {
			return c.Error(err.Error())
		}
		if isPrivateAndLocked(ctx, task.Private) {
			task = redactPrivateTask(task)
		}
		return c.Ok(velo.H{"task": task})
	})

	b.Post("/api/tasks/create", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		var req TaskCreateRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		task, err := createVaultTask(ctx, req)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"task": task})
	})

	b.Post("/api/tasks/update", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		var req TaskUpdateRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		task, err := updateVaultTask(ctx, req)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"task": task})
	})

	b.Post("/api/tasks/complete", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		var req TaskIDRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		task, err := completeVaultTask(ctx, req.ID)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"task": task})
	})

	b.Post("/api/tasks/delete", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		var req TaskIDRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		if err := deleteVaultTask(ctx, req.ID); err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"success": true})
	})

	b.Post("/api/tasks/notes/create", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		var req TaskNoteCreateRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		task, memo, err := createVaultTaskNote(ctx, req)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"memo": memo, "task": task})
	})

	b.Post("/api/tasks/extract-from-memo", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		var req TaskExtractRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		parent, child, memo, err := extractSubtaskFromMemoLine(ctx, req)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"childTask": child, "memo": memo, "parentTask": parent})
	})

	b.Get("/api/task-index/rebuild", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		index, err := rebuildTaskIndex(ctx)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{
			"index": index,
			"tasks": taskIndexEntries(index),
		})
	})
}
