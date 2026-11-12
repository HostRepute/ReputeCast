from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.public_page_language_status import PublicPageLanguageStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.public_copy_labels import PublicCopyLabels





T = TypeVar("T", bound="PublicPageLanguage")



@_attrs_define
class PublicPageLanguage:
    """ 
        Attributes:
            by (str):
            code (str):  Example: es.
            is_default (bool):
            is_fallback (bool):
            is_source (bool):
            labels (PublicCopyLabels):
            missing (int):
            name (str):  Example: Spanish.
            native (str):  Example: Español.
            own (int):
            status (PublicPageLanguageStatus):
            written (int):
            at (datetime.datetime | Unset):
     """

    by: str
    code: str
    is_default: bool
    is_fallback: bool
    is_source: bool
    labels: PublicCopyLabels
    missing: int
    name: str
    native: str
    own: int
    status: PublicPageLanguageStatus
    written: int
    at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.public_copy_labels import PublicCopyLabels # noqa: PLC0415
        by = self.by

        code = self.code

        is_default = self.is_default

        is_fallback = self.is_fallback

        is_source = self.is_source

        labels = self.labels.to_dict()

        missing = self.missing

        name = self.name

        native = self.native

        own = self.own

        status = self.status.value

        written = self.written

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "by": by,
            "code": code,
            "isDefault": is_default,
            "isFallback": is_fallback,
            "isSource": is_source,
            "labels": labels,
            "missing": missing,
            "name": name,
            "native": native,
            "own": own,
            "status": status,
            "written": written,
        })
        if at is not UNSET:
            field_dict["at"] = at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.public_copy_labels import PublicCopyLabels # noqa: PLC0415
        d = dict(src_dict)
        by = d.pop("by")

        code = d.pop("code")

        is_default = d.pop("isDefault")

        is_fallback = d.pop("isFallback")

        is_source = d.pop("isSource")

        labels = PublicCopyLabels.from_dict(d.pop("labels"))




        missing = d.pop("missing")

        name = d.pop("name")

        native = d.pop("native")

        own = d.pop("own")

        status = PublicPageLanguageStatus(d.pop("status"))




        written = d.pop("written")

        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        public_page_language = cls(
            by=by,
            code=code,
            is_default=is_default,
            is_fallback=is_fallback,
            is_source=is_source,
            labels=labels,
            missing=missing,
            name=name,
            native=native,
            own=own,
            status=status,
            written=written,
            at=at,
        )

        return public_page_language

