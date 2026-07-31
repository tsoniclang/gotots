package artifact

import "github.com/tsoniclang/gotots/internal/emit/api"

type artifactOwnerQueue struct {
	owners  []api.ArtifactOwner
	indices map[api.ArtifactOwner]int
	compare func(api.ArtifactOwner, api.ArtifactOwner) int
}

func newArtifactOwnerQueue(
	compare func(api.ArtifactOwner, api.ArtifactOwner) int,
) artifactOwnerQueue {
	return artifactOwnerQueue{
		indices: make(map[api.ArtifactOwner]int),
		compare: compare,
	}
}

func (q *artifactOwnerQueue) push(owner api.ArtifactOwner) {
	if _, exists := q.indices[owner]; exists {
		return
	}
	q.owners = append(q.owners, owner)
	index := len(q.owners) - 1
	q.indices[owner] = index
	q.moveUp(index)
}

func (q *artifactOwnerQueue) pop() (api.ArtifactOwner, bool) {
	if len(q.owners) == 0 {
		return api.ArtifactOwner{}, false
	}
	selected := q.owners[0]
	q.removeAt(0)
	return selected, true
}

func (q *artifactOwnerQueue) discard(owner api.ArtifactOwner) {
	index, exists := q.indices[owner]
	if !exists {
		return
	}
	q.removeAt(index)
}

func (q *artifactOwnerQueue) pending() bool {
	return len(q.owners) != 0
}

func (q *artifactOwnerQueue) removeAt(index int) {
	removed := q.owners[index]
	lastIndex := len(q.owners) - 1
	last := q.owners[lastIndex]
	q.owners = q.owners[:lastIndex]
	delete(q.indices, removed)
	if index == lastIndex {
		return
	}
	q.owners[index] = last
	q.indices[last] = index
	if index > 0 {
		parent := (index - 1) / 2
		if q.compare(q.owners[index], q.owners[parent]) < 0 {
			q.moveUp(index)
			return
		}
	}
	q.moveDown(index)
}

func (q *artifactOwnerQueue) moveUp(index int) {
	for index > 0 {
		parent := (index - 1) / 2
		if q.compare(q.owners[parent], q.owners[index]) <= 0 {
			break
		}
		q.swap(index, parent)
		index = parent
	}
}

func (q *artifactOwnerQueue) moveDown(index int) {
	for {
		left := index*2 + 1
		if left >= len(q.owners) {
			return
		}
		right := left + 1
		selected := left
		if right < len(q.owners) &&
			q.compare(q.owners[right], q.owners[left]) < 0 {
			selected = right
		}
		if q.compare(q.owners[index], q.owners[selected]) <= 0 {
			return
		}
		q.swap(index, selected)
		index = selected
	}
}

func (q *artifactOwnerQueue) swap(left int, right int) {
	q.owners[left], q.owners[right] = q.owners[right], q.owners[left]
	q.indices[q.owners[left]] = left
	q.indices[q.owners[right]] = right
}
