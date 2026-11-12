from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.digest_source_ref_kind import DigestSourceRefKind
from typing import cast






T = TypeVar("T", bound="DigestSourceRef")



@_attrs_define
class DigestSourceRef:
    """ 
        Attributes:
            categories (list[str]):
            kind (DigestSourceRefKind):
            label (str):
     """

    categories: list[str]
    kind: DigestSourceRefKind
    label: str





    def to_dict(self) -> dict[str, Any]:
        categories = self.categories



        kind = self.kind.value

        label = self.label


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "categories": categories,
            "kind": kind,
            "label": label,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        categories = cast(list[str], d.pop("categories"))


        kind = DigestSourceRefKind(d.pop("kind"))




        label = d.pop("label")

        digest_source_ref = cls(
            categories=categories,
            kind=kind,
            label=label,
        )

        return digest_source_ref

