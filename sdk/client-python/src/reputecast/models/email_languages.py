from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.email_language import EmailLanguage
  from ..models.language_source_change import LanguageSourceChange





T = TypeVar("T", bound="EmailLanguages")



@_attrs_define
class EmailLanguages:
    """ 
        Attributes:
            blocks (int):
            can_change_source (bool):
            can_edit (bool):
            can_manage_workspace (bool):
            default (str):
            edit_helpers (list[str]):
            fallback (str):
            source (str):
            source_version (int):
            versions (list[EmailLanguage]):
            workspace_helpers (list[str]):
            readers (int | Unset):
            source_change (LanguageSourceChange | Unset):
     """

    blocks: int
    can_change_source: bool
    can_edit: bool
    can_manage_workspace: bool
    default: str
    edit_helpers: list[str]
    fallback: str
    source: str
    source_version: int
    versions: list[EmailLanguage]
    workspace_helpers: list[str]
    readers: int | Unset = UNSET
    source_change: LanguageSourceChange | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.email_language import EmailLanguage # noqa: PLC0415
        from ..models.language_source_change import LanguageSourceChange # noqa: PLC0415
        blocks = self.blocks

        can_change_source = self.can_change_source

        can_edit = self.can_edit

        can_manage_workspace = self.can_manage_workspace

        default = self.default

        edit_helpers = self.edit_helpers



        fallback = self.fallback

        source = self.source

        source_version = self.source_version

        versions = []
        for versions_item_data in self.versions:
            versions_item = versions_item_data.to_dict()
            versions.append(versions_item)



        workspace_helpers = self.workspace_helpers



        readers = self.readers

        source_change: dict[str, Any] | Unset = UNSET
        if not isinstance(self.source_change, Unset):
            source_change = self.source_change.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blocks": blocks,
            "canChangeSource": can_change_source,
            "canEdit": can_edit,
            "canManageWorkspace": can_manage_workspace,
            "default": default,
            "editHelpers": edit_helpers,
            "fallback": fallback,
            "source": source,
            "sourceVersion": source_version,
            "versions": versions,
            "workspaceHelpers": workspace_helpers,
        })
        if readers is not UNSET:
            field_dict["readers"] = readers
        if source_change is not UNSET:
            field_dict["sourceChange"] = source_change

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.email_language import EmailLanguage # noqa: PLC0415
        from ..models.language_source_change import LanguageSourceChange # noqa: PLC0415
        d = dict(src_dict)
        blocks = d.pop("blocks")

        can_change_source = d.pop("canChangeSource")

        can_edit = d.pop("canEdit")

        can_manage_workspace = d.pop("canManageWorkspace")

        default = d.pop("default")

        edit_helpers = cast(list[str], d.pop("editHelpers"))


        fallback = d.pop("fallback")

        source = d.pop("source")

        source_version = d.pop("sourceVersion")

        versions = []
        _versions = d.pop("versions")
        for versions_item_data in (_versions):
            versions_item = EmailLanguage.from_dict(versions_item_data)



            versions.append(versions_item)


        workspace_helpers = cast(list[str], d.pop("workspaceHelpers"))


        readers = d.pop("readers", UNSET)

        _source_change = d.pop("sourceChange", UNSET)
        source_change: LanguageSourceChange | Unset
        if isinstance(_source_change,  Unset):
            source_change = UNSET
        else:
            source_change = LanguageSourceChange.from_dict(_source_change)




        email_languages = cls(
            blocks=blocks,
            can_change_source=can_change_source,
            can_edit=can_edit,
            can_manage_workspace=can_manage_workspace,
            default=default,
            edit_helpers=edit_helpers,
            fallback=fallback,
            source=source,
            source_version=source_version,
            versions=versions,
            workspace_helpers=workspace_helpers,
            readers=readers,
            source_change=source_change,
        )

        return email_languages

