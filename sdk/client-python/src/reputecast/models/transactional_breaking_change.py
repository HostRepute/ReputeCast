from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_breaking_variable import TransactionalBreakingVariable





T = TypeVar("T", bound="TransactionalBreakingChange")



@_attrs_define
class TransactionalBreakingChange:
    """ 
        Attributes:
            breaking (list[TransactionalBreakingVariable]):
            calls (int): Calls to the template in the last 7 days
            summary (str):
            unused (list[str]):
     """

    breaking: list[TransactionalBreakingVariable]
    calls: int
    summary: str
    unused: list[str]





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_breaking_variable import TransactionalBreakingVariable # noqa: PLC0415
        breaking = []
        for breaking_item_data in self.breaking:
            breaking_item = breaking_item_data.to_dict()
            breaking.append(breaking_item)



        calls = self.calls

        summary = self.summary

        unused = self.unused




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "breaking": breaking,
            "calls": calls,
            "summary": summary,
            "unused": unused,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_breaking_variable import TransactionalBreakingVariable # noqa: PLC0415
        d = dict(src_dict)
        breaking = []
        _breaking = d.pop("breaking")
        for breaking_item_data in (_breaking):
            breaking_item = TransactionalBreakingVariable.from_dict(breaking_item_data)



            breaking.append(breaking_item)


        calls = d.pop("calls")

        summary = d.pop("summary")

        unused = cast(list[str], d.pop("unused"))


        transactional_breaking_change = cls(
            breaking=breaking,
            calls=calls,
            summary=summary,
            unused=unused,
        )

        return transactional_breaking_change

