from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="ReferralSettingsImpact")



@_attrs_define
class ReferralSettingsImpact:
    """ 
        Attributes:
            awaiting (int):
            hold_from (int):
            hold_to (int):
            in_hold (int):
            rules_changed (list[str]):
     """

    awaiting: int
    hold_from: int
    hold_to: int
    in_hold: int
    rules_changed: list[str]





    def to_dict(self) -> dict[str, Any]:
        awaiting = self.awaiting

        hold_from = self.hold_from

        hold_to = self.hold_to

        in_hold = self.in_hold

        rules_changed = self.rules_changed




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "awaiting": awaiting,
            "holdFrom": hold_from,
            "holdTo": hold_to,
            "inHold": in_hold,
            "rulesChanged": rules_changed,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        awaiting = d.pop("awaiting")

        hold_from = d.pop("holdFrom")

        hold_to = d.pop("holdTo")

        in_hold = d.pop("inHold")

        rules_changed = cast(list[str], d.pop("rulesChanged"))


        referral_settings_impact = cls(
            awaiting=awaiting,
            hold_from=hold_from,
            hold_to=hold_to,
            in_hold=in_hold,
            rules_changed=rules_changed,
        )

        return referral_settings_impact

