from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sms_overview_tiles_by_kind import SmsOverviewTilesByKind
  from ..models.sms_overview_tiles_suppressed_by import SmsOverviewTilesSuppressedBy
  from ..models.sms_provider_spend import SmsProviderSpend





T = TypeVar("T", bound="SmsOverviewTiles")



@_attrs_define
class SmsOverviewTiles:
    """ 
        Attributes:
            alert_pct (int):
            budget (float):
            budget_held (int):
            by_kind (SmsOverviewTilesByKind):
            delivered (int):
            delivered_pct (float):
            failed (int):
            month (datetime.datetime):
            sent (int):
            spend (float):
            spend_by (list[SmsProviderSpend]):
            suppressed (int):
            suppressed_by (SmsOverviewTilesSuppressedBy):
     """

    alert_pct: int
    budget: float
    budget_held: int
    by_kind: SmsOverviewTilesByKind
    delivered: int
    delivered_pct: float
    failed: int
    month: datetime.datetime
    sent: int
    spend: float
    spend_by: list[SmsProviderSpend]
    suppressed: int
    suppressed_by: SmsOverviewTilesSuppressedBy





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_overview_tiles_by_kind import SmsOverviewTilesByKind # noqa: PLC0415
        from ..models.sms_overview_tiles_suppressed_by import SmsOverviewTilesSuppressedBy # noqa: PLC0415
        from ..models.sms_provider_spend import SmsProviderSpend # noqa: PLC0415
        alert_pct = self.alert_pct

        budget = self.budget

        budget_held = self.budget_held

        by_kind = self.by_kind.to_dict()

        delivered = self.delivered

        delivered_pct = self.delivered_pct

        failed = self.failed

        month = self.month.isoformat()

        sent = self.sent

        spend = self.spend

        spend_by = []
        for spend_by_item_data in self.spend_by:
            spend_by_item = spend_by_item_data.to_dict()
            spend_by.append(spend_by_item)



        suppressed = self.suppressed

        suppressed_by = self.suppressed_by.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "alertPct": alert_pct,
            "budget": budget,
            "budgetHeld": budget_held,
            "byKind": by_kind,
            "delivered": delivered,
            "deliveredPct": delivered_pct,
            "failed": failed,
            "month": month,
            "sent": sent,
            "spend": spend,
            "spendBy": spend_by,
            "suppressed": suppressed,
            "suppressedBy": suppressed_by,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_overview_tiles_by_kind import SmsOverviewTilesByKind # noqa: PLC0415
        from ..models.sms_overview_tiles_suppressed_by import SmsOverviewTilesSuppressedBy # noqa: PLC0415
        from ..models.sms_provider_spend import SmsProviderSpend # noqa: PLC0415
        d = dict(src_dict)
        alert_pct = d.pop("alertPct")

        budget = d.pop("budget")

        budget_held = d.pop("budgetHeld")

        by_kind = SmsOverviewTilesByKind.from_dict(d.pop("byKind"))




        delivered = d.pop("delivered")

        delivered_pct = d.pop("deliveredPct")

        failed = d.pop("failed")

        month = datetime.datetime.fromisoformat(d.pop("month"))




        sent = d.pop("sent")

        spend = d.pop("spend")

        spend_by = []
        _spend_by = d.pop("spendBy")
        for spend_by_item_data in (_spend_by):
            spend_by_item = SmsProviderSpend.from_dict(spend_by_item_data)



            spend_by.append(spend_by_item)


        suppressed = d.pop("suppressed")

        suppressed_by = SmsOverviewTilesSuppressedBy.from_dict(d.pop("suppressedBy"))




        sms_overview_tiles = cls(
            alert_pct=alert_pct,
            budget=budget,
            budget_held=budget_held,
            by_kind=by_kind,
            delivered=delivered,
            delivered_pct=delivered_pct,
            failed=failed,
            month=month,
            sent=sent,
            spend=spend,
            spend_by=spend_by,
            suppressed=suppressed,
            suppressed_by=suppressed_by,
        )

        return sms_overview_tiles

