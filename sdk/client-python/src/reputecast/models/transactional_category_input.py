from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_category_input_kind import TransactionalCategoryInputKind
from ..types import UNSET, Unset






T = TypeVar("T", bound="TransactionalCategoryInput")



@_attrs_define
class TransactionalCategoryInput:
    """ 
        Attributes:
            kind (TransactionalCategoryInputKind):
            name (str):
            description (str | Unset):
     """

    kind: TransactionalCategoryInputKind
    name: str
    description: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        name = self.name

        description = self.description


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "name": name,
        })
        if description is not UNSET:
            field_dict["description"] = description

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = TransactionalCategoryInputKind(d.pop("kind"))




        name = d.pop("name")

        description = d.pop("description", UNSET)

        transactional_category_input = cls(
            kind=kind,
            name=name,
            description=description,
        )

        return transactional_category_input

