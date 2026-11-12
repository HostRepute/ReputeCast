from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.deliverability_reputation_status import DeliverabilityReputationStatus
from ..models.deliverability_reputation_via import DeliverabilityReputationVia
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.deliverability_reputation_listing import DeliverabilityReputationListing





T = TypeVar("T", bound="DeliverabilityReputation")



@_attrs_define
class DeliverabilityReputation:
    """ 
        Attributes:
            listings (list[DeliverabilityReputationListing]):
            sources (int):
            status (DeliverabilityReputationStatus):
            via (DeliverabilityReputationVia):
            checked_at (datetime.datetime | Unset):
            reason (str | Unset):
     """

    listings: list[DeliverabilityReputationListing]
    sources: int
    status: DeliverabilityReputationStatus
    via: DeliverabilityReputationVia
    checked_at: datetime.datetime | Unset = UNSET
    reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.deliverability_reputation_listing import DeliverabilityReputationListing # noqa: PLC0415
        listings = []
        for listings_item_data in self.listings:
            listings_item = listings_item_data.to_dict()
            listings.append(listings_item)



        sources = self.sources

        status = self.status.value

        via = self.via.value

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "listings": listings,
            "sources": sources,
            "status": status,
            "via": via,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.deliverability_reputation_listing import DeliverabilityReputationListing # noqa: PLC0415
        d = dict(src_dict)
        listings = []
        _listings = d.pop("listings")
        for listings_item_data in (_listings):
            listings_item = DeliverabilityReputationListing.from_dict(listings_item_data)



            listings.append(listings_item)


        sources = d.pop("sources")

        status = DeliverabilityReputationStatus(d.pop("status"))




        via = DeliverabilityReputationVia(d.pop("via"))




        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        reason = d.pop("reason", UNSET)

        deliverability_reputation = cls(
            listings=listings,
            sources=sources,
            status=status,
            via=via,
            checked_at=checked_at,
            reason=reason,
        )

        return deliverability_reputation

