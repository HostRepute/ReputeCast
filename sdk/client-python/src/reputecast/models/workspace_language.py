from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.language_page_copy import LanguagePageCopy





T = TypeVar("T", bound="WorkspaceLanguage")



@_attrs_define
class WorkspaceLanguage:
    """ 
        Attributes:
            added_by (str):
            code (str):  Example: es.
            codes (list[str]):
            contacts (int):
            is_default (bool):
            is_fallback (bool):
            more_codes (bool):
            name (str):  Example: Spanish.
            native (str):  Example: Español.
            own (int):
            page_copy (LanguagePageCopy):
            added_at (datetime.datetime | Unset):
     """

    added_by: str
    code: str
    codes: list[str]
    contacts: int
    is_default: bool
    is_fallback: bool
    more_codes: bool
    name: str
    native: str
    own: int
    page_copy: LanguagePageCopy
    added_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.language_page_copy import LanguagePageCopy # noqa: PLC0415
        added_by = self.added_by

        code = self.code

        codes = self.codes



        contacts = self.contacts

        is_default = self.is_default

        is_fallback = self.is_fallback

        more_codes = self.more_codes

        name = self.name

        native = self.native

        own = self.own

        page_copy = self.page_copy.to_dict()

        added_at: str | Unset = UNSET
        if not isinstance(self.added_at, Unset):
            added_at = self.added_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "addedBy": added_by,
            "code": code,
            "codes": codes,
            "contacts": contacts,
            "isDefault": is_default,
            "isFallback": is_fallback,
            "moreCodes": more_codes,
            "name": name,
            "native": native,
            "own": own,
            "pageCopy": page_copy,
        })
        if added_at is not UNSET:
            field_dict["addedAt"] = added_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.language_page_copy import LanguagePageCopy # noqa: PLC0415
        d = dict(src_dict)
        added_by = d.pop("addedBy")

        code = d.pop("code")

        codes = cast(list[str], d.pop("codes"))


        contacts = d.pop("contacts")

        is_default = d.pop("isDefault")

        is_fallback = d.pop("isFallback")

        more_codes = d.pop("moreCodes")

        name = d.pop("name")

        native = d.pop("native")

        own = d.pop("own")

        page_copy = LanguagePageCopy.from_dict(d.pop("pageCopy"))




        _added_at = d.pop("addedAt", UNSET)
        added_at: datetime.datetime | Unset
        if isinstance(_added_at,  Unset):
            added_at = UNSET
        else:
            added_at = datetime.datetime.fromisoformat(_added_at)




        workspace_language = cls(
            added_by=added_by,
            code=code,
            codes=codes,
            contacts=contacts,
            is_default=is_default,
            is_fallback=is_fallback,
            more_codes=more_codes,
            name=name,
            native=native,
            own=own,
            page_copy=page_copy,
            added_at=added_at,
        )

        return workspace_language

