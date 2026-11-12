from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SmsBudgetView")



@_attrs_define
class SmsBudgetView:
    """ 
        Attributes:
            after (float):
            alert_pct (int):
            budget (float):
            month (datetime.datetime):
            spent (float):
            set_at (datetime.datetime | Unset):
            set_by (str | Unset):
     """

    after: float
    alert_pct: int
    budget: float
    month: datetime.datetime
    spent: float
    set_at: datetime.datetime | Unset = UNSET
    set_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        after = self.after

        alert_pct = self.alert_pct

        budget = self.budget

        month = self.month.isoformat()

        spent = self.spent

        set_at: str | Unset = UNSET
        if not isinstance(self.set_at, Unset):
            set_at = self.set_at.isoformat()

        set_by = self.set_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "after": after,
            "alertPct": alert_pct,
            "budget": budget,
            "month": month,
            "spent": spent,
        })
        if set_at is not UNSET:
            field_dict["setAt"] = set_at
        if set_by is not UNSET:
            field_dict["setBy"] = set_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        after = d.pop("after")

        alert_pct = d.pop("alertPct")

        budget = d.pop("budget")

        month = datetime.datetime.fromisoformat(d.pop("month"))




        spent = d.pop("spent")

        _set_at = d.pop("setAt", UNSET)
        set_at: datetime.datetime | Unset
        if isinstance(_set_at,  Unset):
            set_at = UNSET
        else:
            set_at = datetime.datetime.fromisoformat(_set_at)




        set_by = d.pop("setBy", UNSET)

        sms_budget_view = cls(
            after=after,
            alert_pct=alert_pct,
            budget=budget,
            month=month,
            spent=spent,
            set_at=set_at,
            set_by=set_by,
        )

        return sms_budget_view

