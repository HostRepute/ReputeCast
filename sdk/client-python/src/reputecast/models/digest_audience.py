from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="DigestAudience")



@_attrs_define
class DigestAudience:
    """ 
        Attributes:
            base (int):
            duplicates (int):
            empty (bool):
            final (int):
            label (str):
            list_ids (list[str]):
            role (int):
            segment_ids (list[str]):
            stopped (int):
            suppressed (int):
     """

    base: int
    duplicates: int
    empty: bool
    final: int
    label: str
    list_ids: list[str]
    role: int
    segment_ids: list[str]
    stopped: int
    suppressed: int





    def to_dict(self) -> dict[str, Any]:
        base = self.base

        duplicates = self.duplicates

        empty = self.empty

        final = self.final

        label = self.label

        list_ids = self.list_ids



        role = self.role

        segment_ids = self.segment_ids



        stopped = self.stopped

        suppressed = self.suppressed


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "base": base,
            "duplicates": duplicates,
            "empty": empty,
            "final": final,
            "label": label,
            "listIds": list_ids,
            "role": role,
            "segmentIds": segment_ids,
            "stopped": stopped,
            "suppressed": suppressed,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        base = d.pop("base")

        duplicates = d.pop("duplicates")

        empty = d.pop("empty")

        final = d.pop("final")

        label = d.pop("label")

        list_ids = cast(list[str], d.pop("listIds"))


        role = d.pop("role")

        segment_ids = cast(list[str], d.pop("segmentIds"))


        stopped = d.pop("stopped")

        suppressed = d.pop("suppressed")

        digest_audience = cls(
            base=base,
            duplicates=duplicates,
            empty=empty,
            final=final,
            label=label,
            list_ids=list_ids,
            role=role,
            segment_ids=segment_ids,
            stopped=stopped,
            suppressed=suppressed,
        )

        return digest_audience

