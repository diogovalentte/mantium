package kaizoku

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/diogovalentte/mantium/api/src/util"
)

func (k *Kaizoku) GetQueues() ([]*Queue, error) {
	errorContext := "(kaizoku) error while getting queues"

	url := fmt.Sprintf("%s/bull/queues/api/queues", k.Address)
	resp, err := k.baseRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}
	defer resp.Body.Close()
	err = validateResponse(resp)
	if err != nil {
		return nil, util.AddErrorContext(errorContext, err)
	}

	var queues getQueuesResponse
	err = json.NewDecoder(resp.Body).Decode(&queues)
	if err != nil {
		return nil, util.AddErrorContext(util.AddErrorContext(errorContext, fmt.Errorf("error while decoding response body")).Error(), err)
	}

	return queues.Queues, nil
}

func (k *Kaizoku) GetQueue(queueName string) (*Queue, error) {
	errorContext := "(kaizoku) error while getting queue '%s'"

	queues, err := k.GetQueues()
	if err != nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, queueName), err)
	}

	var queue *Queue
	for _, q := range queues {
		if q.Name == queueName {
			queue = q
			break
		}
	}

	if queue == nil {
		return nil, util.AddErrorContext(fmt.Sprintf(errorContext, queueName), fmt.Errorf("queue not found"))
	}

	return queue, nil
}

func (k *Kaizoku) RetryFailedFixOutOfSyncChaptersQueueJobs() error {
	errorContext := "(kaizoku) error while retrying failed 'fix out of sync chapters' queue jobs"

	url := fmt.Sprintf("%s/bull/queues/api/queues/fixOutOfSyncChaptersQueue/retry/failed", k.Address)
	resp, err := k.baseRequest(http.MethodPut, url, nil)
	if err != nil {
		return util.AddErrorContext(errorContext, err)
	}
	defer resp.Body.Close()
	err = validateResponse(resp)
	if err != nil {
		return util.AddErrorContext(errorContext, err)
	}

	return nil
}

type getQueuesResponse struct {
	Queues []*Queue `json:"queues"`
}
