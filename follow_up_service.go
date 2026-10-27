package main

import (
	"fmt"
	"net/http"
	"strings"
)

type followUp struct {
	Donor  string
	Action string
	Hours  int
}

func delayedSchedule(d followUp) (string, error) {
	if strings.TrimSpace(d.Donor) == "" || d.Hours < 1 {
		return "", fmt.Errorf("donor and positive hours are required")
	}
	return fmt.Sprintf("0 %d * * *", d.Hours), nil
}

func scheduleFollowUp(c *client, d followUp, taskURL string) (string, error) {
	expr, err := delayedSchedule(d)
	if err != nil {
		return "", err
	}
	job, err := c.createCron(cronCreate{CronExpr: expr, Task: taskURL}, "follow-up-"+d.Donor+"-"+d.Action)
	if err != nil {
		return "", err
	}
	return job.JobID, nil
}

func runFollowUp(c *client, d followUp) error {
	if d.Action == "campaign-report" {
		return fmt.Errorf("report destination is unavailable")
	}
	return nil
}

func followUpHandler(c *client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d := followUp{Donor: r.URL.Query().Get("donor"), Action: r.URL.Query().Get("action"), Hours: 1}
		if err := runFollowUp(c, d); err != nil {
			// infrai.errors.capture records the failed delayed action.
			_ = c.captureFailure(err.Error(), []string{"delayed-follow-up", d.Action}, map[string]string{"action": d.Action, "donor": d.Donor})
			http.Error(w, "follow-up recorded", http.StatusAccepted)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func main() {
	c, err := newClient()
	if err != nil {
		panic(err)
	}
	http.Handle("/tasks/follow-up", followUpHandler(c))
	followUpURL := "https://example.org/tasks/follow-up"
	for _, action := range []string{"donor-receipt", "volunteer-reminder", "campaign-report"} {
		id, scheduleErr := scheduleFollowUp(c, followUp{Donor: "demo-donor", Action: action, Hours: 6}, followUpURL+"?action="+action+"&donor=demo-donor")
		if scheduleErr != nil {
			panic(scheduleErr)
		}
		fmt.Println(action, id)
	}
	panic(http.ListenAndServe(":8080", nil))
}
