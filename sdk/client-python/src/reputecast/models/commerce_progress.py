from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="CommerceProgress")



@_attrs_define
class CommerceProgress:
    """ 
        Attributes:
            done (int):
            total (int):
     """

    done: int
    total: int





    def to_dict(self) -> dict[str, Any]:
        done = self.done

        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "done": done,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        done = d.pop("done")

        total = d.pop("total")

        commerce_progress = cls(
            done=done,
            total=total,
        )

        return commerce_progress

