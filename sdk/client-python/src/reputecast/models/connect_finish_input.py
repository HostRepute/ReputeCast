from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_finish_input_source import ConnectFinishInputSource
from ..types import UNSET, Unset






T = TypeVar("T", bound="ConnectFinishInput")



@_attrs_define
class ConnectFinishInput:
    """ 
        Attributes:
            pull_suppressions (bool):
            start_empty (bool):
            source (ConnectFinishInputSource | Unset):
     """

    pull_suppressions: bool
    start_empty: bool
    source: ConnectFinishInputSource | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        pull_suppressions = self.pull_suppressions

        start_empty = self.start_empty

        source: str | Unset = UNSET
        if not isinstance(self.source, Unset):
            source = self.source.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "pullSuppressions": pull_suppressions,
            "startEmpty": start_empty,
        })
        if source is not UNSET:
            field_dict["source"] = source

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        pull_suppressions = d.pop("pullSuppressions")

        start_empty = d.pop("startEmpty")

        _source = d.pop("source", UNSET)
        source: ConnectFinishInputSource | Unset
        if isinstance(_source,  Unset):
            source = UNSET
        else:
            source = ConnectFinishInputSource(_source)




        connect_finish_input = cls(
            pull_suppressions=pull_suppressions,
            start_empty=start_empty,
            source=source,
        )

        return connect_finish_input

