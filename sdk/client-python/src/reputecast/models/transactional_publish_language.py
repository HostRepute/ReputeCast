from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.transactional_publish_language_status import TransactionalPublishLanguageStatus






T = TypeVar("T", bound="TransactionalPublishLanguage")



@_attrs_define
class TransactionalPublishLanguage:
    """ 
        Attributes:
            attention (int):
            gets (str):
            gets_name (str):
            language (str):
            name (str):
            status (TransactionalPublishLanguageStatus):
     """

    attention: int
    gets: str
    gets_name: str
    language: str
    name: str
    status: TransactionalPublishLanguageStatus





    def to_dict(self) -> dict[str, Any]:
        attention = self.attention

        gets = self.gets

        gets_name = self.gets_name

        language = self.language

        name = self.name

        status = self.status.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "attention": attention,
            "gets": gets,
            "getsName": gets_name,
            "language": language,
            "name": name,
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        attention = d.pop("attention")

        gets = d.pop("gets")

        gets_name = d.pop("getsName")

        language = d.pop("language")

        name = d.pop("name")

        status = TransactionalPublishLanguageStatus(d.pop("status"))




        transactional_publish_language = cls(
            attention=attention,
            gets=gets,
            gets_name=gets_name,
            language=language,
            name=name,
            status=status,
        )

        return transactional_publish_language

