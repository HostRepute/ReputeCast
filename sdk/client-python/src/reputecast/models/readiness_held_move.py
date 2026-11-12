from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReadinessHeldMove")



@_attrs_define
class ReadinessHeldMove:
    """ 
        Attributes:
            add_cost (float):
            add_minutes (int):
            from_ (str):
            to (str):
            to_id (str):
     """

    add_cost: float
    add_minutes: int
    from_: str
    to: str
    to_id: str





    def to_dict(self) -> dict[str, Any]:
        add_cost = self.add_cost

        add_minutes = self.add_minutes

        from_ = self.from_

        to = self.to

        to_id = self.to_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "addCost": add_cost,
            "addMinutes": add_minutes,
            "from": from_,
            "to": to,
            "toId": to_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        add_cost = d.pop("addCost")

        add_minutes = d.pop("addMinutes")

        from_ = d.pop("from")

        to = d.pop("to")

        to_id = d.pop("toId")

        readiness_held_move = cls(
            add_cost=add_cost,
            add_minutes=add_minutes,
            from_=from_,
            to=to,
            to_id=to_id,
        )

        return readiness_held_move

