from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.revenue_move import RevenueMove





T = TypeVar("T", bound="RevenueMovement")



@_attrs_define
class RevenueMovement:
    """ 
        Attributes:
            churned (RevenueMove):
            contraction (RevenueMove):
            expansion (RevenueMove):
            from_cents (int):
            from_members (int):
            new (RevenueMove):
            reactivated (RevenueMove):
            to_cents (int):
            to_members (int):
     """

    churned: RevenueMove
    contraction: RevenueMove
    expansion: RevenueMove
    from_cents: int
    from_members: int
    new: RevenueMove
    reactivated: RevenueMove
    to_cents: int
    to_members: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.revenue_move import RevenueMove # noqa: PLC0415
        churned = self.churned.to_dict()

        contraction = self.contraction.to_dict()

        expansion = self.expansion.to_dict()

        from_cents = self.from_cents

        from_members = self.from_members

        new = self.new.to_dict()

        reactivated = self.reactivated.to_dict()

        to_cents = self.to_cents

        to_members = self.to_members


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "churned": churned,
            "contraction": contraction,
            "expansion": expansion,
            "fromCents": from_cents,
            "fromMembers": from_members,
            "new": new,
            "reactivated": reactivated,
            "toCents": to_cents,
            "toMembers": to_members,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.revenue_move import RevenueMove # noqa: PLC0415
        d = dict(src_dict)
        churned = RevenueMove.from_dict(d.pop("churned"))




        contraction = RevenueMove.from_dict(d.pop("contraction"))




        expansion = RevenueMove.from_dict(d.pop("expansion"))




        from_cents = d.pop("fromCents")

        from_members = d.pop("fromMembers")

        new = RevenueMove.from_dict(d.pop("new"))




        reactivated = RevenueMove.from_dict(d.pop("reactivated"))




        to_cents = d.pop("toCents")

        to_members = d.pop("toMembers")

        revenue_movement = cls(
            churned=churned,
            contraction=contraction,
            expansion=expansion,
            from_cents=from_cents,
            from_members=from_members,
            new=new,
            reactivated=reactivated,
            to_cents=to_cents,
            to_members=to_members,
        )

        return revenue_movement

