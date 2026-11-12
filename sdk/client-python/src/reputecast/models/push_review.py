from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.push_review_verdict import PushReviewVerdict
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.push_access import PushAccess
  from ..models.push_audience import PushAudience
  from ..models.push_check import PushCheck
  from ..models.push_guards import PushGuards
  from ..models.push_review_site import PushReviewSite
  from ..models.push_schedule import PushSchedule
  from ..models.push_send_state import PushSendState
  from ..models.push_test_result import PushTestResult
  from ..models.review_approval import ReviewApproval





T = TypeVar("T", bound="PushReview")



@_attrs_define
class PushReview:
    """ 
        Attributes:
            access (PushAccess):
            approval (ReviewApproval):
            audience (PushAudience):
            checked_at (datetime.datetime):
            checks (list[PushCheck]):
            guards (PushGuards):
            id (str):
            name (str):
            schedule (PushSchedule):
            sites (list[PushReviewSite]):
            status (str):
            verdict (PushReviewVerdict):
            version (int):
            send (PushSendState | Unset):
            test (PushTestResult | Unset):
     """

    access: PushAccess
    approval: ReviewApproval
    audience: PushAudience
    checked_at: datetime.datetime
    checks: list[PushCheck]
    guards: PushGuards
    id: str
    name: str
    schedule: PushSchedule
    sites: list[PushReviewSite]
    status: str
    verdict: PushReviewVerdict
    version: int
    send: PushSendState | Unset = UNSET
    test: PushTestResult | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.push_access import PushAccess # noqa: PLC0415
        from ..models.push_audience import PushAudience # noqa: PLC0415
        from ..models.push_check import PushCheck # noqa: PLC0415
        from ..models.push_guards import PushGuards # noqa: PLC0415
        from ..models.push_review_site import PushReviewSite # noqa: PLC0415
        from ..models.push_schedule import PushSchedule # noqa: PLC0415
        from ..models.push_send_state import PushSendState # noqa: PLC0415
        from ..models.push_test_result import PushTestResult # noqa: PLC0415
        from ..models.review_approval import ReviewApproval # noqa: PLC0415
        access = self.access.to_dict()

        approval = self.approval.to_dict()

        audience = self.audience.to_dict()

        checked_at = self.checked_at.isoformat()

        checks = []
        for checks_item_data in self.checks:
            checks_item = checks_item_data.to_dict()
            checks.append(checks_item)



        guards = self.guards.to_dict()

        id = self.id

        name = self.name

        schedule = self.schedule.to_dict()

        sites = []
        for sites_item_data in self.sites:
            sites_item = sites_item_data.to_dict()
            sites.append(sites_item)



        status = self.status

        verdict = self.verdict.value

        version = self.version

        send: dict[str, Any] | Unset = UNSET
        if not isinstance(self.send, Unset):
            send = self.send.to_dict()

        test: dict[str, Any] | Unset = UNSET
        if not isinstance(self.test, Unset):
            test = self.test.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "access": access,
            "approval": approval,
            "audience": audience,
            "checkedAt": checked_at,
            "checks": checks,
            "guards": guards,
            "id": id,
            "name": name,
            "schedule": schedule,
            "sites": sites,
            "status": status,
            "verdict": verdict,
            "version": version,
        })
        if send is not UNSET:
            field_dict["send"] = send
        if test is not UNSET:
            field_dict["test"] = test

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.push_access import PushAccess # noqa: PLC0415
        from ..models.push_audience import PushAudience # noqa: PLC0415
        from ..models.push_check import PushCheck # noqa: PLC0415
        from ..models.push_guards import PushGuards # noqa: PLC0415
        from ..models.push_review_site import PushReviewSite # noqa: PLC0415
        from ..models.push_schedule import PushSchedule # noqa: PLC0415
        from ..models.push_send_state import PushSendState # noqa: PLC0415
        from ..models.push_test_result import PushTestResult # noqa: PLC0415
        from ..models.review_approval import ReviewApproval # noqa: PLC0415
        d = dict(src_dict)
        access = PushAccess.from_dict(d.pop("access"))




        approval = ReviewApproval.from_dict(d.pop("approval"))




        audience = PushAudience.from_dict(d.pop("audience"))




        checked_at = datetime.datetime.fromisoformat(d.pop("checkedAt"))




        checks = []
        _checks = d.pop("checks")
        for checks_item_data in (_checks):
            checks_item = PushCheck.from_dict(checks_item_data)



            checks.append(checks_item)


        guards = PushGuards.from_dict(d.pop("guards"))




        id = d.pop("id")

        name = d.pop("name")

        schedule = PushSchedule.from_dict(d.pop("schedule"))




        sites = []
        _sites = d.pop("sites")
        for sites_item_data in (_sites):
            sites_item = PushReviewSite.from_dict(sites_item_data)



            sites.append(sites_item)


        status = d.pop("status")

        verdict = PushReviewVerdict(d.pop("verdict"))




        version = d.pop("version")

        _send = d.pop("send", UNSET)
        send: PushSendState | Unset
        if isinstance(_send,  Unset):
            send = UNSET
        else:
            send = PushSendState.from_dict(_send)




        _test = d.pop("test", UNSET)
        test: PushTestResult | Unset
        if isinstance(_test,  Unset):
            test = UNSET
        else:
            test = PushTestResult.from_dict(_test)




        push_review = cls(
            access=access,
            approval=approval,
            audience=audience,
            checked_at=checked_at,
            checks=checks,
            guards=guards,
            id=id,
            name=name,
            schedule=schedule,
            sites=sites,
            status=status,
            verdict=verdict,
            version=version,
            send=send,
            test=test,
        )

        return push_review

