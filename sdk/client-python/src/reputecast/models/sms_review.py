from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_review_pending_policy import SmsReviewPendingPolicy
from ..models.sms_review_verdict import SmsReviewVerdict
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.country_line import CountryLine
  from ..models.review_approval import ReviewApproval
  from ..models.sms_budget_view import SmsBudgetView
  from ..models.sms_campaign_access import SmsCampaignAccess
  from ..models.sms_check import SmsCheck
  from ..models.sms_funnel import SmsFunnel
  from ..models.sms_large import SmsLarge
  from ..models.sms_preview import SmsPreview
  from ..models.sms_review_campaign import SmsReviewCampaign
  from ..models.sms_review_policy import SmsReviewPolicy
  from ..models.sms_review_route import SmsReviewRoute
  from ..models.sms_schedule import SmsSchedule
  from ..models.sms_send_view import SmsSendView
  from ..models.sms_test_number import SmsTestNumber
  from ..models.sms_thresholds import SmsThresholds





T = TypeVar("T", bound="SmsReview")



@_attrs_define
class SmsReview:
    """ 
        Attributes:
            access (SmsCampaignAccess):
            approval (ReviewApproval):
            blocks (int):
            budget (SmsBudgetView):
            campaign (SmsReviewCampaign):
            checked_at (datetime.datetime):
            checks (list[SmsCheck]):
            cost (float):
            costs (list[CountryLine]):
            defaults (SmsThresholds):
            fingerprint (str):
            funnel (SmsFunnel):
            large (SmsLarge):
            lists (list[str]):
            passed (int):
            pending_policy (SmsReviewPendingPolicy):
            policy (SmsReviewPolicy):
            preview (SmsPreview):
            routes (list[SmsReviewRoute]):
            schedule (SmsSchedule):
            segments (int):
            segments_each (int):
            tests (list[SmsTestNumber]):
            texts (int):
            thresholds (SmsThresholds):
            timezone (str):
            verdict (SmsReviewVerdict):
            warnings (int):
            priced_on (datetime.datetime | Unset):
            send (SmsSendView | Unset):
     """

    access: SmsCampaignAccess
    approval: ReviewApproval
    blocks: int
    budget: SmsBudgetView
    campaign: SmsReviewCampaign
    checked_at: datetime.datetime
    checks: list[SmsCheck]
    cost: float
    costs: list[CountryLine]
    defaults: SmsThresholds
    fingerprint: str
    funnel: SmsFunnel
    large: SmsLarge
    lists: list[str]
    passed: int
    pending_policy: SmsReviewPendingPolicy
    policy: SmsReviewPolicy
    preview: SmsPreview
    routes: list[SmsReviewRoute]
    schedule: SmsSchedule
    segments: int
    segments_each: int
    tests: list[SmsTestNumber]
    texts: int
    thresholds: SmsThresholds
    timezone: str
    verdict: SmsReviewVerdict
    warnings: int
    priced_on: datetime.datetime | Unset = UNSET
    send: SmsSendView | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.country_line import CountryLine # noqa: PLC0415
        from ..models.review_approval import ReviewApproval # noqa: PLC0415
        from ..models.sms_budget_view import SmsBudgetView # noqa: PLC0415
        from ..models.sms_campaign_access import SmsCampaignAccess # noqa: PLC0415
        from ..models.sms_check import SmsCheck # noqa: PLC0415
        from ..models.sms_funnel import SmsFunnel # noqa: PLC0415
        from ..models.sms_large import SmsLarge # noqa: PLC0415
        from ..models.sms_preview import SmsPreview # noqa: PLC0415
        from ..models.sms_review_campaign import SmsReviewCampaign # noqa: PLC0415
        from ..models.sms_review_policy import SmsReviewPolicy # noqa: PLC0415
        from ..models.sms_review_route import SmsReviewRoute # noqa: PLC0415
        from ..models.sms_schedule import SmsSchedule # noqa: PLC0415
        from ..models.sms_send_view import SmsSendView # noqa: PLC0415
        from ..models.sms_test_number import SmsTestNumber # noqa: PLC0415
        from ..models.sms_thresholds import SmsThresholds # noqa: PLC0415
        access = self.access.to_dict()

        approval = self.approval.to_dict()

        blocks = self.blocks

        budget = self.budget.to_dict()

        campaign = self.campaign.to_dict()

        checked_at = self.checked_at.isoformat()

        checks = []
        for checks_item_data in self.checks:
            checks_item = checks_item_data.to_dict()
            checks.append(checks_item)



        cost = self.cost

        costs = []
        for costs_item_data in self.costs:
            costs_item = costs_item_data.to_dict()
            costs.append(costs_item)



        defaults = self.defaults.to_dict()

        fingerprint = self.fingerprint

        funnel = self.funnel.to_dict()

        large = self.large.to_dict()

        lists = self.lists



        passed = self.passed

        pending_policy = self.pending_policy.value

        policy = self.policy.to_dict()

        preview = self.preview.to_dict()

        routes = []
        for routes_item_data in self.routes:
            routes_item = routes_item_data.to_dict()
            routes.append(routes_item)



        schedule = self.schedule.to_dict()

        segments = self.segments

        segments_each = self.segments_each

        tests = []
        for tests_item_data in self.tests:
            tests_item = tests_item_data.to_dict()
            tests.append(tests_item)



        texts = self.texts

        thresholds = self.thresholds.to_dict()

        timezone = self.timezone

        verdict = self.verdict.value

        warnings = self.warnings

        priced_on: str | Unset = UNSET
        if not isinstance(self.priced_on, Unset):
            priced_on = self.priced_on.isoformat()

        send: dict[str, Any] | Unset = UNSET
        if not isinstance(self.send, Unset):
            send = self.send.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "access": access,
            "approval": approval,
            "blocks": blocks,
            "budget": budget,
            "campaign": campaign,
            "checkedAt": checked_at,
            "checks": checks,
            "cost": cost,
            "costs": costs,
            "defaults": defaults,
            "fingerprint": fingerprint,
            "funnel": funnel,
            "large": large,
            "lists": lists,
            "passed": passed,
            "pendingPolicy": pending_policy,
            "policy": policy,
            "preview": preview,
            "routes": routes,
            "schedule": schedule,
            "segments": segments,
            "segmentsEach": segments_each,
            "tests": tests,
            "texts": texts,
            "thresholds": thresholds,
            "timezone": timezone,
            "verdict": verdict,
            "warnings": warnings,
        })
        if priced_on is not UNSET:
            field_dict["pricedOn"] = priced_on
        if send is not UNSET:
            field_dict["send"] = send

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.country_line import CountryLine # noqa: PLC0415
        from ..models.review_approval import ReviewApproval # noqa: PLC0415
        from ..models.sms_budget_view import SmsBudgetView # noqa: PLC0415
        from ..models.sms_campaign_access import SmsCampaignAccess # noqa: PLC0415
        from ..models.sms_check import SmsCheck # noqa: PLC0415
        from ..models.sms_funnel import SmsFunnel # noqa: PLC0415
        from ..models.sms_large import SmsLarge # noqa: PLC0415
        from ..models.sms_preview import SmsPreview # noqa: PLC0415
        from ..models.sms_review_campaign import SmsReviewCampaign # noqa: PLC0415
        from ..models.sms_review_policy import SmsReviewPolicy # noqa: PLC0415
        from ..models.sms_review_route import SmsReviewRoute # noqa: PLC0415
        from ..models.sms_schedule import SmsSchedule # noqa: PLC0415
        from ..models.sms_send_view import SmsSendView # noqa: PLC0415
        from ..models.sms_test_number import SmsTestNumber # noqa: PLC0415
        from ..models.sms_thresholds import SmsThresholds # noqa: PLC0415
        d = dict(src_dict)
        access = SmsCampaignAccess.from_dict(d.pop("access"))




        approval = ReviewApproval.from_dict(d.pop("approval"))




        blocks = d.pop("blocks")

        budget = SmsBudgetView.from_dict(d.pop("budget"))




        campaign = SmsReviewCampaign.from_dict(d.pop("campaign"))




        checked_at = datetime.datetime.fromisoformat(d.pop("checkedAt"))




        checks = []
        _checks = d.pop("checks")
        for checks_item_data in (_checks):
            checks_item = SmsCheck.from_dict(checks_item_data)



            checks.append(checks_item)


        cost = d.pop("cost")

        costs = []
        _costs = d.pop("costs")
        for costs_item_data in (_costs):
            costs_item = CountryLine.from_dict(costs_item_data)



            costs.append(costs_item)


        defaults = SmsThresholds.from_dict(d.pop("defaults"))




        fingerprint = d.pop("fingerprint")

        funnel = SmsFunnel.from_dict(d.pop("funnel"))




        large = SmsLarge.from_dict(d.pop("large"))




        lists = cast(list[str], d.pop("lists"))


        passed = d.pop("passed")

        pending_policy = SmsReviewPendingPolicy(d.pop("pendingPolicy"))




        policy = SmsReviewPolicy.from_dict(d.pop("policy"))




        preview = SmsPreview.from_dict(d.pop("preview"))




        routes = []
        _routes = d.pop("routes")
        for routes_item_data in (_routes):
            routes_item = SmsReviewRoute.from_dict(routes_item_data)



            routes.append(routes_item)


        schedule = SmsSchedule.from_dict(d.pop("schedule"))




        segments = d.pop("segments")

        segments_each = d.pop("segmentsEach")

        tests = []
        _tests = d.pop("tests")
        for tests_item_data in (_tests):
            tests_item = SmsTestNumber.from_dict(tests_item_data)



            tests.append(tests_item)


        texts = d.pop("texts")

        thresholds = SmsThresholds.from_dict(d.pop("thresholds"))




        timezone = d.pop("timezone")

        verdict = SmsReviewVerdict(d.pop("verdict"))




        warnings = d.pop("warnings")

        _priced_on = d.pop("pricedOn", UNSET)
        priced_on: datetime.datetime | Unset
        if isinstance(_priced_on,  Unset):
            priced_on = UNSET
        else:
            priced_on = datetime.datetime.fromisoformat(_priced_on)




        _send = d.pop("send", UNSET)
        send: SmsSendView | Unset
        if isinstance(_send,  Unset):
            send = UNSET
        else:
            send = SmsSendView.from_dict(_send)




        sms_review = cls(
            access=access,
            approval=approval,
            blocks=blocks,
            budget=budget,
            campaign=campaign,
            checked_at=checked_at,
            checks=checks,
            cost=cost,
            costs=costs,
            defaults=defaults,
            fingerprint=fingerprint,
            funnel=funnel,
            large=large,
            lists=lists,
            passed=passed,
            pending_policy=pending_policy,
            policy=policy,
            preview=preview,
            routes=routes,
            schedule=schedule,
            segments=segments,
            segments_each=segments_each,
            tests=tests,
            texts=texts,
            thresholds=thresholds,
            timezone=timezone,
            verdict=verdict,
            warnings=warnings,
            priced_on=priced_on,
            send=send,
        )

        return sms_review

