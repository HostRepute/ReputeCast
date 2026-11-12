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
  from ..models.grow_helper_sets import GrowHelperSets
  from ..models.referral_filter_option import ReferralFilterOption
  from ..models.referral_review_group import ReferralReviewGroup
  from ..models.referral_signal_count import ReferralSignalCount





T = TypeVar("T", bound="ReferralReviewQueue")



@_attrs_define
class ReferralReviewQueue:
    """ 
        Attributes:
            groups (list[ReferralReviewGroup]):
            held (int):
            helpers (GrowHelperSets):
            hold_days (int):
            referrers (list[ReferralFilterOption]):
            show_addresses (bool):
            signals (list[ReferralSignalCount]):
            oldest (datetime.datetime | Unset):
     """

    groups: list[ReferralReviewGroup]
    held: int
    helpers: GrowHelperSets
    hold_days: int
    referrers: list[ReferralFilterOption]
    show_addresses: bool
    signals: list[ReferralSignalCount]
    oldest: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.grow_helper_sets import GrowHelperSets # noqa: PLC0415
        from ..models.referral_filter_option import ReferralFilterOption # noqa: PLC0415
        from ..models.referral_review_group import ReferralReviewGroup # noqa: PLC0415
        from ..models.referral_signal_count import ReferralSignalCount # noqa: PLC0415
        groups = []
        for groups_item_data in self.groups:
            groups_item = groups_item_data.to_dict()
            groups.append(groups_item)



        held = self.held

        helpers = self.helpers.to_dict()

        hold_days = self.hold_days

        referrers = []
        for referrers_item_data in self.referrers:
            referrers_item = referrers_item_data.to_dict()
            referrers.append(referrers_item)



        show_addresses = self.show_addresses

        signals = []
        for signals_item_data in self.signals:
            signals_item = signals_item_data.to_dict()
            signals.append(signals_item)



        oldest: str | Unset = UNSET
        if not isinstance(self.oldest, Unset):
            oldest = self.oldest.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "groups": groups,
            "held": held,
            "helpers": helpers,
            "holdDays": hold_days,
            "referrers": referrers,
            "showAddresses": show_addresses,
            "signals": signals,
        })
        if oldest is not UNSET:
            field_dict["oldest"] = oldest

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.grow_helper_sets import GrowHelperSets # noqa: PLC0415
        from ..models.referral_filter_option import ReferralFilterOption # noqa: PLC0415
        from ..models.referral_review_group import ReferralReviewGroup # noqa: PLC0415
        from ..models.referral_signal_count import ReferralSignalCount # noqa: PLC0415
        d = dict(src_dict)
        groups = []
        _groups = d.pop("groups")
        for groups_item_data in (_groups):
            groups_item = ReferralReviewGroup.from_dict(groups_item_data)



            groups.append(groups_item)


        held = d.pop("held")

        helpers = GrowHelperSets.from_dict(d.pop("helpers"))




        hold_days = d.pop("holdDays")

        referrers = []
        _referrers = d.pop("referrers")
        for referrers_item_data in (_referrers):
            referrers_item = ReferralFilterOption.from_dict(referrers_item_data)



            referrers.append(referrers_item)


        show_addresses = d.pop("showAddresses")

        signals = []
        _signals = d.pop("signals")
        for signals_item_data in (_signals):
            signals_item = ReferralSignalCount.from_dict(signals_item_data)



            signals.append(signals_item)


        _oldest = d.pop("oldest", UNSET)
        oldest: datetime.datetime | Unset
        if isinstance(_oldest,  Unset):
            oldest = UNSET
        else:
            oldest = datetime.datetime.fromisoformat(_oldest)




        referral_review_queue = cls(
            groups=groups,
            held=held,
            helpers=helpers,
            hold_days=hold_days,
            referrers=referrers,
            show_addresses=show_addresses,
            signals=signals,
            oldest=oldest,
        )

        return referral_review_queue

