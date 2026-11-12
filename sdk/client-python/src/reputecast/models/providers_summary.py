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
  from ..models.provider_month_amount import ProviderMonthAmount
  from ..models.provider_status_count import ProviderStatusCount





T = TypeVar("T", bound="ProvidersSummary")



@_attrs_define
class ProvidersSummary:
    """ 
        Attributes:
            by_status (list[ProviderStatusCount]):
            connected (int):
            cost_to_date (float): Dollars
            in_sync (bool):
            month_start (datetime.datetime):
            pushed (int):
            sent_by (list[ProviderMonthAmount]):
            sent_this_month (int):
            suppressions (int):
            checked_at (datetime.datetime | Unset):
            cost_per_1000 (float | Unset):
            synced_at (datetime.datetime | Unset):
     """

    by_status: list[ProviderStatusCount]
    connected: int
    cost_to_date: float
    in_sync: bool
    month_start: datetime.datetime
    pushed: int
    sent_by: list[ProviderMonthAmount]
    sent_this_month: int
    suppressions: int
    checked_at: datetime.datetime | Unset = UNSET
    cost_per_1000: float | Unset = UNSET
    synced_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.provider_month_amount import ProviderMonthAmount # noqa: PLC0415
        from ..models.provider_status_count import ProviderStatusCount # noqa: PLC0415
        by_status = []
        for by_status_item_data in self.by_status:
            by_status_item = by_status_item_data.to_dict()
            by_status.append(by_status_item)



        connected = self.connected

        cost_to_date = self.cost_to_date

        in_sync = self.in_sync

        month_start = self.month_start.isoformat()

        pushed = self.pushed

        sent_by = []
        for sent_by_item_data in self.sent_by:
            sent_by_item = sent_by_item_data.to_dict()
            sent_by.append(sent_by_item)



        sent_this_month = self.sent_this_month

        suppressions = self.suppressions

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()

        cost_per_1000 = self.cost_per_1000

        synced_at: str | Unset = UNSET
        if not isinstance(self.synced_at, Unset):
            synced_at = self.synced_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "byStatus": by_status,
            "connected": connected,
            "costToDate": cost_to_date,
            "inSync": in_sync,
            "monthStart": month_start,
            "pushed": pushed,
            "sentBy": sent_by,
            "sentThisMonth": sent_this_month,
            "suppressions": suppressions,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at
        if cost_per_1000 is not UNSET:
            field_dict["costPer1000"] = cost_per_1000
        if synced_at is not UNSET:
            field_dict["syncedAt"] = synced_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.provider_month_amount import ProviderMonthAmount # noqa: PLC0415
        from ..models.provider_status_count import ProviderStatusCount # noqa: PLC0415
        d = dict(src_dict)
        by_status = []
        _by_status = d.pop("byStatus")
        for by_status_item_data in (_by_status):
            by_status_item = ProviderStatusCount.from_dict(by_status_item_data)



            by_status.append(by_status_item)


        connected = d.pop("connected")

        cost_to_date = d.pop("costToDate")

        in_sync = d.pop("inSync")

        month_start = datetime.datetime.fromisoformat(d.pop("monthStart"))




        pushed = d.pop("pushed")

        sent_by = []
        _sent_by = d.pop("sentBy")
        for sent_by_item_data in (_sent_by):
            sent_by_item = ProviderMonthAmount.from_dict(sent_by_item_data)



            sent_by.append(sent_by_item)


        sent_this_month = d.pop("sentThisMonth")

        suppressions = d.pop("suppressions")

        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        cost_per_1000 = d.pop("costPer1000", UNSET)

        _synced_at = d.pop("syncedAt", UNSET)
        synced_at: datetime.datetime | Unset
        if isinstance(_synced_at,  Unset):
            synced_at = UNSET
        else:
            synced_at = datetime.datetime.fromisoformat(_synced_at)




        providers_summary = cls(
            by_status=by_status,
            connected=connected,
            cost_to_date=cost_to_date,
            in_sync=in_sync,
            month_start=month_start,
            pushed=pushed,
            sent_by=sent_by,
            sent_this_month=sent_this_month,
            suppressions=suppressions,
            checked_at=checked_at,
            cost_per_1000=cost_per_1000,
            synced_at=synced_at,
        )

        return providers_summary

