from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="EarnMemberCounts")



@_attrs_define
class EarnMemberCounts:
    """ 
        Attributes:
            cancels_soon (int):
            complimentary (int):
            ended (int):
            gift (int):
            past_due (int):
            paying (int):
            trialing (int):
            with_access (int):
     """

    cancels_soon: int
    complimentary: int
    ended: int
    gift: int
    past_due: int
    paying: int
    trialing: int
    with_access: int





    def to_dict(self) -> dict[str, Any]:
        cancels_soon = self.cancels_soon

        complimentary = self.complimentary

        ended = self.ended

        gift = self.gift

        past_due = self.past_due

        paying = self.paying

        trialing = self.trialing

        with_access = self.with_access


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cancelsSoon": cancels_soon,
            "complimentary": complimentary,
            "ended": ended,
            "gift": gift,
            "pastDue": past_due,
            "paying": paying,
            "trialing": trialing,
            "withAccess": with_access,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cancels_soon = d.pop("cancelsSoon")

        complimentary = d.pop("complimentary")

        ended = d.pop("ended")

        gift = d.pop("gift")

        past_due = d.pop("pastDue")

        paying = d.pop("paying")

        trialing = d.pop("trialing")

        with_access = d.pop("withAccess")

        earn_member_counts = cls(
            cancels_soon=cancels_soon,
            complimentary=complimentary,
            ended=ended,
            gift=gift,
            past_due=past_due,
            paying=paying,
            trialing=trialing,
            with_access=with_access,
        )

        return earn_member_counts

