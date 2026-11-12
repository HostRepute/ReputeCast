from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.placement_content_ref_kind import PlacementContentRefKind






T = TypeVar("T", bound="PlacementContentRef")



@_attrs_define
class PlacementContentRef:
    """ 
        Attributes:
            id (str): A campaign, template, recurring digest (its next issue) or transactional template id
            kind (PlacementContentRefKind):
     """

    id: str
    kind: PlacementContentRefKind





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        kind = self.kind.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "kind": kind,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        kind = PlacementContentRefKind(d.pop("kind"))




        placement_content_ref = cls(
            id=id,
            kind=kind,
        )

        return placement_content_ref

