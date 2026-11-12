from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_whats_app_var_source import AutomationWhatsAppVarSource
from ..types import UNSET, Unset






T = TypeVar("T", bound="AutomationWhatsAppVar")



@_attrs_define
class AutomationWhatsAppVar:
    """ 
        Attributes:
            key (str):
            source (AutomationWhatsAppVarSource):
            fallback (str | Unset):
            field (str | Unset):
            text (str | Unset):
     """

    key: str
    source: AutomationWhatsAppVarSource
    fallback: str | Unset = UNSET
    field: str | Unset = UNSET
    text: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        source = self.source.value

        fallback = self.fallback

        field = self.field

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "source": source,
        })
        if fallback is not UNSET:
            field_dict["fallback"] = fallback
        if field is not UNSET:
            field_dict["field"] = field
        if text is not UNSET:
            field_dict["text"] = text

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        source = AutomationWhatsAppVarSource(d.pop("source"))




        fallback = d.pop("fallback", UNSET)

        field = d.pop("field", UNSET)

        text = d.pop("text", UNSET)

        automation_whats_app_var = cls(
            key=key,
            source=source,
            fallback=fallback,
            field=field,
            text=text,
        )

        return automation_whats_app_var

