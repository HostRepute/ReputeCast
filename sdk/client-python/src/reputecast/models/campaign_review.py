from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.readiness_fix_batch import ReadinessFixBatch
  from ..models.readiness_progress import ReadinessProgress
  from ..models.readiness_run import ReadinessRun
  from ..models.readiness_run_failure import ReadinessRunFailure
  from ..models.readiness_schedule import ReadinessSchedule
  from ..models.readiness_thresholds import ReadinessThresholds
  from ..models.review_access import ReviewAccess
  from ..models.review_approval import ReviewApproval
  from ..models.review_campaign import ReviewCampaign
  from ..models.review_canary import ReviewCanary
  from ..models.review_held import ReviewHeld
  from ..models.review_scheduled import ReviewScheduled
  from ..models.review_test_send import ReviewTestSend
  from ..models.review_typed import ReviewTyped





T = TypeVar("T", bound="CampaignReview")



@_attrs_define
class CampaignReview:
    """ 
        Attributes:
            access (ReviewAccess):
            approval (ReviewApproval):
            campaign (ReviewCampaign):
            canary (ReviewCanary):
            pause_slack (bool):
            schedule (ReadinessSchedule):
            send_now_within (int):
            stale (bool):
            test_sends (list[ReviewTestSend]):
            thresholds (ReadinessThresholds):
            thresholds_own (bool):
            timezone (str):
            typed (ReviewTyped):
            failed (ReadinessRunFailure | Unset):
            fixes (ReadinessFixBatch | Unset):
            held (ReviewHeld | Unset):
            run (ReadinessRun | Unset):
            running (ReadinessProgress | Unset):
            scheduled (ReviewScheduled | Unset):
     """

    access: ReviewAccess
    approval: ReviewApproval
    campaign: ReviewCampaign
    canary: ReviewCanary
    pause_slack: bool
    schedule: ReadinessSchedule
    send_now_within: int
    stale: bool
    test_sends: list[ReviewTestSend]
    thresholds: ReadinessThresholds
    thresholds_own: bool
    timezone: str
    typed: ReviewTyped
    failed: ReadinessRunFailure | Unset = UNSET
    fixes: ReadinessFixBatch | Unset = UNSET
    held: ReviewHeld | Unset = UNSET
    run: ReadinessRun | Unset = UNSET
    running: ReadinessProgress | Unset = UNSET
    scheduled: ReviewScheduled | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.readiness_fix_batch import ReadinessFixBatch # noqa: PLC0415
        from ..models.readiness_progress import ReadinessProgress # noqa: PLC0415
        from ..models.readiness_run import ReadinessRun # noqa: PLC0415
        from ..models.readiness_run_failure import ReadinessRunFailure # noqa: PLC0415
        from ..models.readiness_schedule import ReadinessSchedule # noqa: PLC0415
        from ..models.readiness_thresholds import ReadinessThresholds # noqa: PLC0415
        from ..models.review_access import ReviewAccess # noqa: PLC0415
        from ..models.review_approval import ReviewApproval # noqa: PLC0415
        from ..models.review_campaign import ReviewCampaign # noqa: PLC0415
        from ..models.review_canary import ReviewCanary # noqa: PLC0415
        from ..models.review_held import ReviewHeld # noqa: PLC0415
        from ..models.review_scheduled import ReviewScheduled # noqa: PLC0415
        from ..models.review_test_send import ReviewTestSend # noqa: PLC0415
        from ..models.review_typed import ReviewTyped # noqa: PLC0415
        access = self.access.to_dict()

        approval = self.approval.to_dict()

        campaign = self.campaign.to_dict()

        canary = self.canary.to_dict()

        pause_slack = self.pause_slack

        schedule = self.schedule.to_dict()

        send_now_within = self.send_now_within

        stale = self.stale

        test_sends = []
        for test_sends_item_data in self.test_sends:
            test_sends_item = test_sends_item_data.to_dict()
            test_sends.append(test_sends_item)



        thresholds = self.thresholds.to_dict()

        thresholds_own = self.thresholds_own

        timezone = self.timezone

        typed = self.typed.to_dict()

        failed: dict[str, Any] | Unset = UNSET
        if not isinstance(self.failed, Unset):
            failed = self.failed.to_dict()

        fixes: dict[str, Any] | Unset = UNSET
        if not isinstance(self.fixes, Unset):
            fixes = self.fixes.to_dict()

        held: dict[str, Any] | Unset = UNSET
        if not isinstance(self.held, Unset):
            held = self.held.to_dict()

        run: dict[str, Any] | Unset = UNSET
        if not isinstance(self.run, Unset):
            run = self.run.to_dict()

        running: dict[str, Any] | Unset = UNSET
        if not isinstance(self.running, Unset):
            running = self.running.to_dict()

        scheduled: dict[str, Any] | Unset = UNSET
        if not isinstance(self.scheduled, Unset):
            scheduled = self.scheduled.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "access": access,
            "approval": approval,
            "campaign": campaign,
            "canary": canary,
            "pauseSlack": pause_slack,
            "schedule": schedule,
            "sendNowWithin": send_now_within,
            "stale": stale,
            "testSends": test_sends,
            "thresholds": thresholds,
            "thresholdsOwn": thresholds_own,
            "timezone": timezone,
            "typed": typed,
        })
        if failed is not UNSET:
            field_dict["failed"] = failed
        if fixes is not UNSET:
            field_dict["fixes"] = fixes
        if held is not UNSET:
            field_dict["held"] = held
        if run is not UNSET:
            field_dict["run"] = run
        if running is not UNSET:
            field_dict["running"] = running
        if scheduled is not UNSET:
            field_dict["scheduled"] = scheduled

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.readiness_fix_batch import ReadinessFixBatch # noqa: PLC0415
        from ..models.readiness_progress import ReadinessProgress # noqa: PLC0415
        from ..models.readiness_run import ReadinessRun # noqa: PLC0415
        from ..models.readiness_run_failure import ReadinessRunFailure # noqa: PLC0415
        from ..models.readiness_schedule import ReadinessSchedule # noqa: PLC0415
        from ..models.readiness_thresholds import ReadinessThresholds # noqa: PLC0415
        from ..models.review_access import ReviewAccess # noqa: PLC0415
        from ..models.review_approval import ReviewApproval # noqa: PLC0415
        from ..models.review_campaign import ReviewCampaign # noqa: PLC0415
        from ..models.review_canary import ReviewCanary # noqa: PLC0415
        from ..models.review_held import ReviewHeld # noqa: PLC0415
        from ..models.review_scheduled import ReviewScheduled # noqa: PLC0415
        from ..models.review_test_send import ReviewTestSend # noqa: PLC0415
        from ..models.review_typed import ReviewTyped # noqa: PLC0415
        d = dict(src_dict)
        access = ReviewAccess.from_dict(d.pop("access"))




        approval = ReviewApproval.from_dict(d.pop("approval"))




        campaign = ReviewCampaign.from_dict(d.pop("campaign"))




        canary = ReviewCanary.from_dict(d.pop("canary"))




        pause_slack = d.pop("pauseSlack")

        schedule = ReadinessSchedule.from_dict(d.pop("schedule"))




        send_now_within = d.pop("sendNowWithin")

        stale = d.pop("stale")

        test_sends = []
        _test_sends = d.pop("testSends")
        for test_sends_item_data in (_test_sends):
            test_sends_item = ReviewTestSend.from_dict(test_sends_item_data)



            test_sends.append(test_sends_item)


        thresholds = ReadinessThresholds.from_dict(d.pop("thresholds"))




        thresholds_own = d.pop("thresholdsOwn")

        timezone = d.pop("timezone")

        typed = ReviewTyped.from_dict(d.pop("typed"))




        _failed = d.pop("failed", UNSET)
        failed: ReadinessRunFailure | Unset
        if isinstance(_failed,  Unset):
            failed = UNSET
        else:
            failed = ReadinessRunFailure.from_dict(_failed)




        _fixes = d.pop("fixes", UNSET)
        fixes: ReadinessFixBatch | Unset
        if isinstance(_fixes,  Unset):
            fixes = UNSET
        else:
            fixes = ReadinessFixBatch.from_dict(_fixes)




        _held = d.pop("held", UNSET)
        held: ReviewHeld | Unset
        if isinstance(_held,  Unset):
            held = UNSET
        else:
            held = ReviewHeld.from_dict(_held)




        _run = d.pop("run", UNSET)
        run: ReadinessRun | Unset
        if isinstance(_run,  Unset):
            run = UNSET
        else:
            run = ReadinessRun.from_dict(_run)




        _running = d.pop("running", UNSET)
        running: ReadinessProgress | Unset
        if isinstance(_running,  Unset):
            running = UNSET
        else:
            running = ReadinessProgress.from_dict(_running)




        _scheduled = d.pop("scheduled", UNSET)
        scheduled: ReviewScheduled | Unset
        if isinstance(_scheduled,  Unset):
            scheduled = UNSET
        else:
            scheduled = ReviewScheduled.from_dict(_scheduled)




        campaign_review = cls(
            access=access,
            approval=approval,
            campaign=campaign,
            canary=canary,
            pause_slack=pause_slack,
            schedule=schedule,
            send_now_within=send_now_within,
            stale=stale,
            test_sends=test_sends,
            thresholds=thresholds,
            thresholds_own=thresholds_own,
            timezone=timezone,
            typed=typed,
            failed=failed,
            fixes=fixes,
            held=held,
            run=run,
            running=running,
            scheduled=scheduled,
        )

        return campaign_review

