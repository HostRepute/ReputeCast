from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PollCommentCounts")



@_attrs_define
class PollCommentCounts:
    """ 
        Attributes:
            all_ (int):
            detractors (int):
            passives (int):
            promoters (int):
     """

    all_: int
    detractors: int
    passives: int
    promoters: int





    def to_dict(self) -> dict[str, Any]:
        all_ = self.all_

        detractors = self.detractors

        passives = self.passives

        promoters = self.promoters


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "all": all_,
            "detractors": detractors,
            "passives": passives,
            "promoters": promoters,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        all_ = d.pop("all")

        detractors = d.pop("detractors")

        passives = d.pop("passives")

        promoters = d.pop("promoters")

        poll_comment_counts = cls(
            all_=all_,
            detractors=detractors,
            passives=passives,
            promoters=promoters,
        )

        return poll_comment_counts

