from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.whats_app_review_over_limit import WhatsAppReviewOverLimit
from ..models.whats_app_review_verdict import WhatsAppReviewVerdict
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.review_approval import ReviewApproval
  from ..models.whats_app_access import WhatsAppAccess
  from ..models.whats_app_audience import WhatsAppAudience
  from ..models.whats_app_check import WhatsAppCheck
  from ..models.whats_app_cost import WhatsAppCost
  from ..models.whats_app_schedule import WhatsAppSchedule
  from ..models.whats_app_send_state import WhatsAppSendState





T = TypeVar("T", bound="WhatsAppReview")



@_attrs_define
class WhatsAppReview:
    """ 
        Attributes:
            access (WhatsAppAccess):
            approval (ReviewApproval):
            audience (WhatsAppAudience):
            checked_at (datetime.datetime):
            checks (list[WhatsAppCheck]):
            cost (WhatsAppCost):
            id (str):
            name (str):
            over_limit (WhatsAppReviewOverLimit):
            schedule (WhatsAppSchedule):
            status (str):
            verdict (WhatsAppReviewVerdict):
            version (int):
            send (WhatsAppSendState | Unset):
     """

    access: WhatsAppAccess
    approval: ReviewApproval
    audience: WhatsAppAudience
    checked_at: datetime.datetime
    checks: list[WhatsAppCheck]
    cost: WhatsAppCost
    id: str
    name: str
    over_limit: WhatsAppReviewOverLimit
    schedule: WhatsAppSchedule
    status: str
    verdict: WhatsAppReviewVerdict
    version: int
    send: WhatsAppSendState | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.review_approval import ReviewApproval # noqa: PLC0415
        from ..models.whats_app_access import WhatsAppAccess # noqa: PLC0415
        from ..models.whats_app_audience import WhatsAppAudience # noqa: PLC0415
        from ..models.whats_app_check import WhatsAppCheck # noqa: PLC0415
        from ..models.whats_app_cost import WhatsAppCost # noqa: PLC0415
        from ..models.whats_app_schedule import WhatsAppSchedule # noqa: PLC0415
        from ..models.whats_app_send_state import WhatsAppSendState # noqa: PLC0415
        access = self.access.to_dict()

        approval = self.approval.to_dict()

        audience = self.audience.to_dict()

        checked_at = self.checked_at.isoformat()

        checks = []
        for checks_item_data in self.checks:
            checks_item = checks_item_data.to_dict()
            checks.append(checks_item)



        cost = self.cost.to_dict()

        id = self.id

        name = self.name

        over_limit = self.over_limit.value

        schedule = self.schedule.to_dict()

        status = self.status

        verdict = self.verdict.value

        version = self.version

        send: dict[str, Any] | Unset = UNSET
        if not isinstance(self.send, Unset):
            send = self.send.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "access": access,
            "approval": approval,
            "audience": audience,
            "checkedAt": checked_at,
            "checks": checks,
            "cost": cost,
            "id": id,
            "name": name,
            "overLimit": over_limit,
            "schedule": schedule,
            "status": status,
            "verdict": verdict,
            "version": version,
        })
        if send is not UNSET:
            field_dict["send"] = send

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.review_approval import ReviewApproval # noqa: PLC0415
        from ..models.whats_app_access import WhatsAppAccess # noqa: PLC0415
        from ..models.whats_app_audience import WhatsAppAudience # noqa: PLC0415
        from ..models.whats_app_check import WhatsAppCheck # noqa: PLC0415
        from ..models.whats_app_cost import WhatsAppCost # noqa: PLC0415
        from ..models.whats_app_schedule import WhatsAppSchedule # noqa: PLC0415
        from ..models.whats_app_send_state import WhatsAppSendState # noqa: PLC0415
        d = dict(src_dict)
        access = WhatsAppAccess.from_dict(d.pop("access"))




        approval = ReviewApproval.from_dict(d.pop("approval"))




        audience = WhatsAppAudience.from_dict(d.pop("audience"))




        checked_at = datetime.datetime.fromisoformat(d.pop("checkedAt"))




        checks = []
        _checks = d.pop("checks")
        for checks_item_data in (_checks):
            checks_item = WhatsAppCheck.from_dict(checks_item_data)



            checks.append(checks_item)


        cost = WhatsAppCost.from_dict(d.pop("cost"))




        id = d.pop("id")

        name = d.pop("name")

        over_limit = WhatsAppReviewOverLimit(d.pop("overLimit"))




        schedule = WhatsAppSchedule.from_dict(d.pop("schedule"))




        status = d.pop("status")

        verdict = WhatsAppReviewVerdict(d.pop("verdict"))




        version = d.pop("version")

        _send = d.pop("send", UNSET)
        send: WhatsAppSendState | Unset
        if isinstance(_send,  Unset):
            send = UNSET
        else:
            send = WhatsAppSendState.from_dict(_send)




        whats_app_review = cls(
            access=access,
            approval=approval,
            audience=audience,
            checked_at=checked_at,
            checks=checks,
            cost=cost,
            id=id,
            name=name,
            over_limit=over_limit,
            schedule=schedule,
            status=status,
            verdict=verdict,
            version=version,
            send=send,
        )

        return whats_app_review

