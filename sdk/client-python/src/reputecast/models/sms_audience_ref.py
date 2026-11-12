from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_audience_ref_kind import SmsAudienceRefKind






T = TypeVar("T", bound="SmsAudienceRef")



@_attrs_define
class SmsAudienceRef:
    """ 
        Attributes:
            id (str):
            kind (SmsAudienceRefKind):
            name (str):
     """

    id: str
    kind: SmsAudienceRefKind
    name: str





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        kind = self.kind.value

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "kind": kind,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        kind = SmsAudienceRefKind(d.pop("kind"))




        name = d.pop("name")

        sms_audience_ref = cls(
            id=id,
            kind=kind,
            name=name,
        )

        return sms_audience_ref

