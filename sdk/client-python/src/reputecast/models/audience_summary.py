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
  from ..models.audience_list_summary import AudienceListSummary
  from ..models.audience_paid_tier import AudiencePaidTier
  from ..models.audience_tier_counts import AudienceTierCounts





T = TypeVar("T", bound="AudienceSummary")



@_attrs_define
class AudienceSummary:
    """ 
        Attributes:
            contacts (int):
            lists (list[AudienceListSummary]):
            paid_tiers (list[AudiencePaidTier]):
            segments (int):
            stripe (bool):
            suppressions (int):
            tiers (AudienceTierCounts):
            tiers_computed_at (datetime.datetime | Unset):
     """

    contacts: int
    lists: list[AudienceListSummary]
    paid_tiers: list[AudiencePaidTier]
    segments: int
    stripe: bool
    suppressions: int
    tiers: AudienceTierCounts
    tiers_computed_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.audience_list_summary import AudienceListSummary # noqa: PLC0415
        from ..models.audience_paid_tier import AudiencePaidTier # noqa: PLC0415
        from ..models.audience_tier_counts import AudienceTierCounts # noqa: PLC0415
        contacts = self.contacts

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        paid_tiers = []
        for paid_tiers_item_data in self.paid_tiers:
            paid_tiers_item = paid_tiers_item_data.to_dict()
            paid_tiers.append(paid_tiers_item)



        segments = self.segments

        stripe = self.stripe

        suppressions = self.suppressions

        tiers = self.tiers.to_dict()

        tiers_computed_at: str | Unset = UNSET
        if not isinstance(self.tiers_computed_at, Unset):
            tiers_computed_at = self.tiers_computed_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contacts": contacts,
            "lists": lists,
            "paidTiers": paid_tiers,
            "segments": segments,
            "stripe": stripe,
            "suppressions": suppressions,
            "tiers": tiers,
        })
        if tiers_computed_at is not UNSET:
            field_dict["tiersComputedAt"] = tiers_computed_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audience_list_summary import AudienceListSummary # noqa: PLC0415
        from ..models.audience_paid_tier import AudiencePaidTier # noqa: PLC0415
        from ..models.audience_tier_counts import AudienceTierCounts # noqa: PLC0415
        d = dict(src_dict)
        contacts = d.pop("contacts")

        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = AudienceListSummary.from_dict(lists_item_data)



            lists.append(lists_item)


        paid_tiers = []
        _paid_tiers = d.pop("paidTiers")
        for paid_tiers_item_data in (_paid_tiers):
            paid_tiers_item = AudiencePaidTier.from_dict(paid_tiers_item_data)



            paid_tiers.append(paid_tiers_item)


        segments = d.pop("segments")

        stripe = d.pop("stripe")

        suppressions = d.pop("suppressions")

        tiers = AudienceTierCounts.from_dict(d.pop("tiers"))




        _tiers_computed_at = d.pop("tiersComputedAt", UNSET)
        tiers_computed_at: datetime.datetime | Unset
        if isinstance(_tiers_computed_at,  Unset):
            tiers_computed_at = UNSET
        else:
            tiers_computed_at = datetime.datetime.fromisoformat(_tiers_computed_at)




        audience_summary = cls(
            contacts=contacts,
            lists=lists,
            paid_tiers=paid_tiers,
            segments=segments,
            stripe=stripe,
            suppressions=suppressions,
            tiers=tiers,
            tiers_computed_at=tiers_computed_at,
        )

        return audience_summary

