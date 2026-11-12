from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_category_kind import TransactionalCategoryKind






T = TypeVar("T", bound="TransactionalCategory")



@_attrs_define
class TransactionalCategory:
    """ 
        Attributes:
            description (str): What recipients see in the preference center
            id (str):
            kind (TransactionalCategoryKind):
            name (str):
            opted_out (int):
            templates (int):
     """

    description: str
    id: str
    kind: TransactionalCategoryKind
    name: str
    opted_out: int
    templates: int





    def to_dict(self) -> dict[str, Any]:
        description = self.description

        id = self.id

        kind = self.kind.value

        name = self.name

        opted_out = self.opted_out

        templates = self.templates


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "description": description,
            "id": id,
            "kind": kind,
            "name": name,
            "optedOut": opted_out,
            "templates": templates,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        description = d.pop("description")

        id = d.pop("id")

        kind = TransactionalCategoryKind(d.pop("kind"))




        name = d.pop("name")

        opted_out = d.pop("optedOut")

        templates = d.pop("templates")

        transactional_category = cls(
            description=description,
            id=id,
            kind=kind,
            name=name,
            opted_out=opted_out,
            templates=templates,
        )

        return transactional_category

