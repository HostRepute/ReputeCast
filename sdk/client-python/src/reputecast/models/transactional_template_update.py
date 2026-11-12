from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="TransactionalTemplateUpdate")



@_attrs_define
class TransactionalTemplateUpdate:
    """ 
        Attributes:
            category_id (str | Unset):
            description (str | Unset):
            key (str | Unset):
            name (str | Unset):
     """

    category_id: str | Unset = UNSET
    description: str | Unset = UNSET
    key: str | Unset = UNSET
    name: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        category_id = self.category_id

        description = self.description

        key = self.key

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if category_id is not UNSET:
            field_dict["categoryId"] = category_id
        if description is not UNSET:
            field_dict["description"] = description
        if key is not UNSET:
            field_dict["key"] = key
        if name is not UNSET:
            field_dict["name"] = name

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        category_id = d.pop("categoryId", UNSET)

        description = d.pop("description", UNSET)

        key = d.pop("key", UNSET)

        name = d.pop("name", UNSET)

        transactional_template_update = cls(
            category_id=category_id,
            description=description,
            key=key,
            name=name,
        )

        return transactional_template_update

