from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.compare_text_state import CompareTextState






T = TypeVar("T", bound="CompareText")



@_attrs_define
class CompareText:
    """ 
        Attributes:
            before (str):
            source (str):
            state (CompareTextState):
            version (str):
     """

    before: str
    source: str
    state: CompareTextState
    version: str





    def to_dict(self) -> dict[str, Any]:
        before = self.before

        source = self.source

        state = self.state.value

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "before": before,
            "source": source,
            "state": state,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        before = d.pop("before")

        source = d.pop("source")

        state = CompareTextState(d.pop("state"))




        version = d.pop("version")

        compare_text = cls(
            before=before,
            source=source,
            state=state,
            version=version,
        )

        return compare_text

