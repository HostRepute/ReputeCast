from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sender_domain_cell_tone import SenderDomainCellTone






T = TypeVar("T", bound="SenderDomainCell")



@_attrs_define
class SenderDomainCell:
    """ 
        Attributes:
            text (str):
            tone (SenderDomainCellTone):
     """

    text: str
    tone: SenderDomainCellTone





    def to_dict(self) -> dict[str, Any]:
        text = self.text

        tone = self.tone.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "text": text,
            "tone": tone,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        text = d.pop("text")

        tone = SenderDomainCellTone(d.pop("tone"))




        sender_domain_cell = cls(
            text=text,
            tone=tone,
        )

        return sender_domain_cell

