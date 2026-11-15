from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="BrandVoiceContent")



@_attrs_define
class BrandVoiceContent:
    """ 
        Attributes:
            address (str):
            avoided_words (str):
            examples (str):
            preferred_words (str):
            sentence_length (str):
            tone (str):
     """

    address: str
    avoided_words: str
    examples: str
    preferred_words: str
    sentence_length: str
    tone: str





    def to_dict(self) -> dict[str, Any]:
        address = self.address

        avoided_words = self.avoided_words

        examples = self.examples

        preferred_words = self.preferred_words

        sentence_length = self.sentence_length

        tone = self.tone


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "address": address,
            "avoidedWords": avoided_words,
            "examples": examples,
            "preferredWords": preferred_words,
            "sentenceLength": sentence_length,
            "tone": tone,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        address = d.pop("address")

        avoided_words = d.pop("avoidedWords")

        examples = d.pop("examples")

        preferred_words = d.pop("preferredWords")

        sentence_length = d.pop("sentenceLength")

        tone = d.pop("tone")

        brand_voice_content = cls(
            address=address,
            avoided_words=avoided_words,
            examples=examples,
            preferred_words=preferred_words,
            sentence_length=sentence_length,
            tone=tone,
        )

        return brand_voice_content

