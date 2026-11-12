from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.verification_form_count import VerificationFormCount
  from ..models.verification_realtime_day import VerificationRealtimeDay





T = TypeVar("T", bound="VerificationRealtime")



@_attrs_define
class VerificationRealtime:
    """ 
        Attributes:
            asked_to_fix (int):
            checks (int):
            days (list[VerificationRealtimeDay]):
            forms (list[VerificationFormCount]):
            service (str):
            spent (float):
            total (VerificationFormCount):
            waiting (int):
            since (datetime.datetime | Unset):
     """

    asked_to_fix: int
    checks: int
    days: list[VerificationRealtimeDay]
    forms: list[VerificationFormCount]
    service: str
    spent: float
    total: VerificationFormCount
    waiting: int
    since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.verification_form_count import VerificationFormCount # noqa: PLC0415
        from ..models.verification_realtime_day import VerificationRealtimeDay # noqa: PLC0415
        asked_to_fix = self.asked_to_fix

        checks = self.checks

        days = []
        for days_item_data in self.days:
            days_item = days_item_data.to_dict()
            days.append(days_item)



        forms = []
        for forms_item_data in self.forms:
            forms_item = forms_item_data.to_dict()
            forms.append(forms_item)



        service = self.service

        spent = self.spent

        total = self.total.to_dict()

        waiting = self.waiting

        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "askedToFix": asked_to_fix,
            "checks": checks,
            "days": days,
            "forms": forms,
            "service": service,
            "spent": spent,
            "total": total,
            "waiting": waiting,
        })
        if since is not UNSET:
            field_dict["since"] = since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.verification_form_count import VerificationFormCount # noqa: PLC0415
        from ..models.verification_realtime_day import VerificationRealtimeDay # noqa: PLC0415
        d = dict(src_dict)
        asked_to_fix = d.pop("askedToFix")

        checks = d.pop("checks")

        days = []
        _days = d.pop("days")
        for days_item_data in (_days):
            days_item = VerificationRealtimeDay.from_dict(days_item_data)



            days.append(days_item)


        forms = []
        _forms = d.pop("forms")
        for forms_item_data in (_forms):
            forms_item = VerificationFormCount.from_dict(forms_item_data)



            forms.append(forms_item)


        service = d.pop("service")

        spent = d.pop("spent")

        total = VerificationFormCount.from_dict(d.pop("total"))




        waiting = d.pop("waiting")

        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        verification_realtime = cls(
            asked_to_fix=asked_to_fix,
            checks=checks,
            days=days,
            forms=forms,
            service=service,
            spent=spent,
            total=total,
            waiting=waiting,
            since=since,
        )

        return verification_realtime

