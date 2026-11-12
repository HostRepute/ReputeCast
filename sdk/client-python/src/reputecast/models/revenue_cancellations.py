from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.revenue_reason import RevenueReason





T = TypeVar("T", bound="RevenueCancellations")



@_attrs_define
class RevenueCancellations:
    """ 
        Attributes:
            continued (int):
            kept (int):
            reasons (list[RevenueReason]):
            undecided (int):
            visits (int):
            kept_offer (str | Unset):
            kept_tier (str | Unset):
            offer_id (str | Unset):
     """

    continued: int
    kept: int
    reasons: list[RevenueReason]
    undecided: int
    visits: int
    kept_offer: str | Unset = UNSET
    kept_tier: str | Unset = UNSET
    offer_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.revenue_reason import RevenueReason # noqa: PLC0415
        continued = self.continued

        kept = self.kept

        reasons = []
        for reasons_item_data in self.reasons:
            reasons_item = reasons_item_data.to_dict()
            reasons.append(reasons_item)



        undecided = self.undecided

        visits = self.visits

        kept_offer = self.kept_offer

        kept_tier = self.kept_tier

        offer_id = self.offer_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "continued": continued,
            "kept": kept,
            "reasons": reasons,
            "undecided": undecided,
            "visits": visits,
        })
        if kept_offer is not UNSET:
            field_dict["keptOffer"] = kept_offer
        if kept_tier is not UNSET:
            field_dict["keptTier"] = kept_tier
        if offer_id is not UNSET:
            field_dict["offerId"] = offer_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.revenue_reason import RevenueReason # noqa: PLC0415
        d = dict(src_dict)
        continued = d.pop("continued")

        kept = d.pop("kept")

        reasons = []
        _reasons = d.pop("reasons")
        for reasons_item_data in (_reasons):
            reasons_item = RevenueReason.from_dict(reasons_item_data)



            reasons.append(reasons_item)


        undecided = d.pop("undecided")

        visits = d.pop("visits")

        kept_offer = d.pop("keptOffer", UNSET)

        kept_tier = d.pop("keptTier", UNSET)

        offer_id = d.pop("offerId", UNSET)

        revenue_cancellations = cls(
            continued=continued,
            kept=kept,
            reasons=reasons,
            undecided=undecided,
            visits=visits,
            kept_offer=kept_offer,
            kept_tier=kept_tier,
            offer_id=offer_id,
        )

        return revenue_cancellations

