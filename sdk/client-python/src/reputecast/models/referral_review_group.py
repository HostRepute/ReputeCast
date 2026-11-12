from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_review_group_suggestion import ReferralReviewGroupSuggestion
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.referral_held_item import ReferralHeldItem
  from ..models.referral_person import ReferralPerson
  from ..models.referral_signal import ReferralSignal





T = TypeVar("T", bound="ReferralReviewGroup")



@_attrs_define
class ReferralReviewGroup:
    """ 
        Attributes:
            confirmed_days (list[datetime.datetime]):
            days (int):
            different_names (bool):
            engaged (bool):
            hold_ends (list[datetime.datetime]):
            items (list[ReferralHeldItem]):
            opened (int):
            opened_in (str):
            referrer (ReferralPerson):
            same_domain (str):
            signals (list[ReferralSignal]):
            suggestion (ReferralReviewGroupSuggestion):
     """

    confirmed_days: list[datetime.datetime]
    days: int
    different_names: bool
    engaged: bool
    hold_ends: list[datetime.datetime]
    items: list[ReferralHeldItem]
    opened: int
    opened_in: str
    referrer: ReferralPerson
    same_domain: str
    signals: list[ReferralSignal]
    suggestion: ReferralReviewGroupSuggestion





    def to_dict(self) -> dict[str, Any]:
        from ..models.referral_held_item import ReferralHeldItem # noqa: PLC0415
        from ..models.referral_person import ReferralPerson # noqa: PLC0415
        from ..models.referral_signal import ReferralSignal # noqa: PLC0415
        confirmed_days = []
        for confirmed_days_item_data in self.confirmed_days:
            confirmed_days_item = confirmed_days_item_data.isoformat()
            confirmed_days.append(confirmed_days_item)



        days = self.days

        different_names = self.different_names

        engaged = self.engaged

        hold_ends = []
        for hold_ends_item_data in self.hold_ends:
            hold_ends_item = hold_ends_item_data.isoformat()
            hold_ends.append(hold_ends_item)



        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        opened = self.opened

        opened_in = self.opened_in

        referrer = self.referrer.to_dict()

        same_domain = self.same_domain

        signals = []
        for signals_item_data in self.signals:
            signals_item = signals_item_data.to_dict()
            signals.append(signals_item)



        suggestion = self.suggestion.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "confirmedDays": confirmed_days,
            "days": days,
            "differentNames": different_names,
            "engaged": engaged,
            "holdEnds": hold_ends,
            "items": items,
            "opened": opened,
            "openedIn": opened_in,
            "referrer": referrer,
            "sameDomain": same_domain,
            "signals": signals,
            "suggestion": suggestion,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.referral_held_item import ReferralHeldItem # noqa: PLC0415
        from ..models.referral_person import ReferralPerson # noqa: PLC0415
        from ..models.referral_signal import ReferralSignal # noqa: PLC0415
        d = dict(src_dict)
        confirmed_days = []
        _confirmed_days = d.pop("confirmedDays")
        for confirmed_days_item_data in (_confirmed_days):
            confirmed_days_item = datetime.datetime.fromisoformat(confirmed_days_item_data)



            confirmed_days.append(confirmed_days_item)


        days = d.pop("days")

        different_names = d.pop("differentNames")

        engaged = d.pop("engaged")

        hold_ends = []
        _hold_ends = d.pop("holdEnds")
        for hold_ends_item_data in (_hold_ends):
            hold_ends_item = datetime.datetime.fromisoformat(hold_ends_item_data)



            hold_ends.append(hold_ends_item)


        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = ReferralHeldItem.from_dict(items_item_data)



            items.append(items_item)


        opened = d.pop("opened")

        opened_in = d.pop("openedIn")

        referrer = ReferralPerson.from_dict(d.pop("referrer"))




        same_domain = d.pop("sameDomain")

        signals = []
        _signals = d.pop("signals")
        for signals_item_data in (_signals):
            signals_item = ReferralSignal.from_dict(signals_item_data)



            signals.append(signals_item)


        suggestion = ReferralReviewGroupSuggestion(d.pop("suggestion"))




        referral_review_group = cls(
            confirmed_days=confirmed_days,
            days=days,
            different_names=different_names,
            engaged=engaged,
            hold_ends=hold_ends,
            items=items,
            opened=opened,
            opened_in=opened_in,
            referrer=referrer,
            same_domain=same_domain,
            signals=signals,
            suggestion=suggestion,
        )

        return referral_review_group

