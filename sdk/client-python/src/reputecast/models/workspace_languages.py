from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.workspace_language import WorkspaceLanguage





T = TypeVar("T", bound="WorkspaceLanguages")



@_attrs_define
class WorkspaceLanguages:
    """ 
        Attributes:
            can_manage (bool):
            default (str):
            fallback (str):
            helpers (list[str]):
            languages (list[WorkspaceLanguage]):
            no_language (int):
            not_offered (int):
            not_offered_codes (list[str]):
            total_contacts (int):
     """

    can_manage: bool
    default: str
    fallback: str
    helpers: list[str]
    languages: list[WorkspaceLanguage]
    no_language: int
    not_offered: int
    not_offered_codes: list[str]
    total_contacts: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.workspace_language import WorkspaceLanguage # noqa: PLC0415
        can_manage = self.can_manage

        default = self.default

        fallback = self.fallback

        helpers = self.helpers



        languages = []
        for languages_item_data in self.languages:
            languages_item = languages_item_data.to_dict()
            languages.append(languages_item)



        no_language = self.no_language

        not_offered = self.not_offered

        not_offered_codes = self.not_offered_codes



        total_contacts = self.total_contacts


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "default": default,
            "fallback": fallback,
            "helpers": helpers,
            "languages": languages,
            "noLanguage": no_language,
            "notOffered": not_offered,
            "notOfferedCodes": not_offered_codes,
            "totalContacts": total_contacts,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.workspace_language import WorkspaceLanguage # noqa: PLC0415
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        default = d.pop("default")

        fallback = d.pop("fallback")

        helpers = cast(list[str], d.pop("helpers"))


        languages = []
        _languages = d.pop("languages")
        for languages_item_data in (_languages):
            languages_item = WorkspaceLanguage.from_dict(languages_item_data)



            languages.append(languages_item)


        no_language = d.pop("noLanguage")

        not_offered = d.pop("notOffered")

        not_offered_codes = cast(list[str], d.pop("notOfferedCodes"))


        total_contacts = d.pop("totalContacts")

        workspace_languages = cls(
            can_manage=can_manage,
            default=default,
            fallback=fallback,
            helpers=helpers,
            languages=languages,
            no_language=no_language,
            not_offered=not_offered,
            not_offered_codes=not_offered_codes,
            total_contacts=total_contacts,
        )

        return workspace_languages

