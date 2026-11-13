from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReplyCounts")



@_attrs_define
class ReplyCounts:
    """ 
        Attributes:
            auto (int):
            done (int):
            open_ (int):
            other (int):
            spam (int):
            unsubscribes (int):
     """

    auto: int
    done: int
    open_: int
    other: int
    spam: int
    unsubscribes: int





    def to_dict(self) -> dict[str, Any]:
        auto = self.auto

        done = self.done

        open_ = self.open_

        other = self.other

        spam = self.spam

        unsubscribes = self.unsubscribes


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "auto": auto,
            "done": done,
            "open": open_,
            "other": other,
            "spam": spam,
            "unsubscribes": unsubscribes,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        auto = d.pop("auto")

        done = d.pop("done")

        open_ = d.pop("open")

        other = d.pop("other")

        spam = d.pop("spam")

        unsubscribes = d.pop("unsubscribes")

        reply_counts = cls(
            auto=auto,
            done=done,
            open_=open_,
            other=other,
            spam=spam,
            unsubscribes=unsubscribes,
        )

        return reply_counts

