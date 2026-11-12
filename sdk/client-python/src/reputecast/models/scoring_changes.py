from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="ScoringChanges")



@_attrs_define
class ScoringChanges:
    """ 
        Attributes:
            bands (bool):
            decay (bool):
            rules (list[int]):
            setup (bool):
     """

    bands: bool
    decay: bool
    rules: list[int]
    setup: bool





    def to_dict(self) -> dict[str, Any]:
        bands = self.bands

        decay = self.decay

        rules = self.rules



        setup = self.setup


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bands": bands,
            "decay": decay,
            "rules": rules,
            "setup": setup,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        bands = d.pop("bands")

        decay = d.pop("decay")

        rules = cast(list[int], d.pop("rules"))


        setup = d.pop("setup")

        scoring_changes = cls(
            bands=bands,
            decay=decay,
            rules=rules,
            setup=setup,
        )

        return scoring_changes

