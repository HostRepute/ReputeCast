from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.grow_helper_sets import GrowHelperSets
  from ..models.referral_milestone_cnt import ReferralMilestoneCnt
  from ..models.referral_referrer_row import ReferralReferrerRow





T = TypeVar("T", bound="ReferralReferrerList")



@_attrs_define
class ReferralReferrerList:
    """ 
        Attributes:
            held_referrers (int):
            helpers (GrowHelperSets):
            items (list[ReferralReferrerRow]):
            milestone_counts (list[ReferralMilestoneCnt]):
            page (int):
            paused_links (int):
            show_addresses (bool):
            total (int):
            with_counted (int):
     """

    held_referrers: int
    helpers: GrowHelperSets
    items: list[ReferralReferrerRow]
    milestone_counts: list[ReferralMilestoneCnt]
    page: int
    paused_links: int
    show_addresses: bool
    total: int
    with_counted: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.grow_helper_sets import GrowHelperSets # noqa: PLC0415
        from ..models.referral_milestone_cnt import ReferralMilestoneCnt # noqa: PLC0415
        from ..models.referral_referrer_row import ReferralReferrerRow # noqa: PLC0415
        held_referrers = self.held_referrers

        helpers = self.helpers.to_dict()

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        milestone_counts = []
        for milestone_counts_item_data in self.milestone_counts:
            milestone_counts_item = milestone_counts_item_data.to_dict()
            milestone_counts.append(milestone_counts_item)



        page = self.page

        paused_links = self.paused_links

        show_addresses = self.show_addresses

        total = self.total

        with_counted = self.with_counted


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "heldReferrers": held_referrers,
            "helpers": helpers,
            "items": items,
            "milestoneCounts": milestone_counts,
            "page": page,
            "pausedLinks": paused_links,
            "showAddresses": show_addresses,
            "total": total,
            "withCounted": with_counted,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.grow_helper_sets import GrowHelperSets # noqa: PLC0415
        from ..models.referral_milestone_cnt import ReferralMilestoneCnt # noqa: PLC0415
        from ..models.referral_referrer_row import ReferralReferrerRow # noqa: PLC0415
        d = dict(src_dict)
        held_referrers = d.pop("heldReferrers")

        helpers = GrowHelperSets.from_dict(d.pop("helpers"))




        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = ReferralReferrerRow.from_dict(items_item_data)



            items.append(items_item)


        milestone_counts = []
        _milestone_counts = d.pop("milestoneCounts")
        for milestone_counts_item_data in (_milestone_counts):
            milestone_counts_item = ReferralMilestoneCnt.from_dict(milestone_counts_item_data)



            milestone_counts.append(milestone_counts_item)


        page = d.pop("page")

        paused_links = d.pop("pausedLinks")

        show_addresses = d.pop("showAddresses")

        total = d.pop("total")

        with_counted = d.pop("withCounted")

        referral_referrer_list = cls(
            held_referrers=held_referrers,
            helpers=helpers,
            items=items,
            milestone_counts=milestone_counts,
            page=page,
            paused_links=paused_links,
            show_addresses=show_addresses,
            total=total,
            with_counted=with_counted,
        )

        return referral_referrer_list

