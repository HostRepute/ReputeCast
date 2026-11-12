from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_diff_line import TransactionalDiffLine
  from ..models.transactional_language_change import TransactionalLanguageChange
  from ..models.transactional_text_change import TransactionalTextChange
  from ..models.transactional_variable_change import TransactionalVariableChange





T = TypeVar("T", bound="TransactionalDiff")



@_attrs_define
class TransactionalDiff:
    """ 
        Attributes:
            changed (bool):
            content (list[TransactionalDiffLine]):
            content_summary (str):
            settings (list[str]):
            variables (list[TransactionalVariableChange]):
            languages (list[TransactionalLanguageChange] | Unset):
            preheader (TransactionalTextChange | Unset):
            subject (TransactionalTextChange | Unset):
     """

    changed: bool
    content: list[TransactionalDiffLine]
    content_summary: str
    settings: list[str]
    variables: list[TransactionalVariableChange]
    languages: list[TransactionalLanguageChange] | Unset = UNSET
    preheader: TransactionalTextChange | Unset = UNSET
    subject: TransactionalTextChange | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_diff_line import TransactionalDiffLine # noqa: PLC0415
        from ..models.transactional_language_change import TransactionalLanguageChange # noqa: PLC0415
        from ..models.transactional_text_change import TransactionalTextChange # noqa: PLC0415
        from ..models.transactional_variable_change import TransactionalVariableChange # noqa: PLC0415
        changed = self.changed

        content = []
        for content_item_data in self.content:
            content_item = content_item_data.to_dict()
            content.append(content_item)



        content_summary = self.content_summary

        settings = self.settings



        variables = []
        for variables_item_data in self.variables:
            variables_item = variables_item_data.to_dict()
            variables.append(variables_item)



        languages: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.languages, Unset):
            languages = []
            for languages_item_data in self.languages:
                languages_item = languages_item_data.to_dict()
                languages.append(languages_item)



        preheader: dict[str, Any] | Unset = UNSET
        if not isinstance(self.preheader, Unset):
            preheader = self.preheader.to_dict()

        subject: dict[str, Any] | Unset = UNSET
        if not isinstance(self.subject, Unset):
            subject = self.subject.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "changed": changed,
            "content": content,
            "contentSummary": content_summary,
            "settings": settings,
            "variables": variables,
        })
        if languages is not UNSET:
            field_dict["languages"] = languages
        if preheader is not UNSET:
            field_dict["preheader"] = preheader
        if subject is not UNSET:
            field_dict["subject"] = subject

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_diff_line import TransactionalDiffLine # noqa: PLC0415
        from ..models.transactional_language_change import TransactionalLanguageChange # noqa: PLC0415
        from ..models.transactional_text_change import TransactionalTextChange # noqa: PLC0415
        from ..models.transactional_variable_change import TransactionalVariableChange # noqa: PLC0415
        d = dict(src_dict)
        changed = d.pop("changed")

        content = []
        _content = d.pop("content")
        for content_item_data in (_content):
            content_item = TransactionalDiffLine.from_dict(content_item_data)



            content.append(content_item)


        content_summary = d.pop("contentSummary")

        settings = cast(list[str], d.pop("settings"))


        variables = []
        _variables = d.pop("variables")
        for variables_item_data in (_variables):
            variables_item = TransactionalVariableChange.from_dict(variables_item_data)



            variables.append(variables_item)


        _languages = d.pop("languages", UNSET)
        languages: list[TransactionalLanguageChange] | Unset = UNSET
        if _languages is not UNSET:
            languages = []
            for languages_item_data in _languages:
                languages_item = TransactionalLanguageChange.from_dict(languages_item_data)



                languages.append(languages_item)


        _preheader = d.pop("preheader", UNSET)
        preheader: TransactionalTextChange | Unset
        if isinstance(_preheader,  Unset):
            preheader = UNSET
        else:
            preheader = TransactionalTextChange.from_dict(_preheader)




        _subject = d.pop("subject", UNSET)
        subject: TransactionalTextChange | Unset
        if isinstance(_subject,  Unset):
            subject = UNSET
        else:
            subject = TransactionalTextChange.from_dict(_subject)




        transactional_diff = cls(
            changed=changed,
            content=content,
            content_summary=content_summary,
            settings=settings,
            variables=variables,
            languages=languages,
            preheader=preheader,
            subject=subject,
        )

        return transactional_diff

