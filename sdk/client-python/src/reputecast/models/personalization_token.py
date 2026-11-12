from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.personalization_token_kind import PersonalizationTokenKind
from ..types import UNSET, Unset






T = TypeVar("T", bound="PersonalizationToken")



@_attrs_define
class PersonalizationToken:
    """ 
        Attributes:
            key (str):  Example: first_name.
            kind (PersonalizationTokenKind):
            label (str):  Example: First name.
            fallback (str | Unset):
     """

    key: str
    kind: PersonalizationTokenKind
    label: str
    fallback: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        kind = self.kind.value

        label = self.label

        fallback = self.fallback


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "kind": kind,
            "label": label,
        })
        if fallback is not UNSET:
            field_dict["fallback"] = fallback

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        kind = PersonalizationTokenKind(d.pop("kind"))




        label = d.pop("label")

        fallback = d.pop("fallback", UNSET)

        personalization_token = cls(
            key=key,
            kind=kind,
            label=label,
            fallback=fallback,
        )

        return personalization_token

