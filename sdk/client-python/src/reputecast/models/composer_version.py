from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.composer_version_source import ComposerVersionSource
from typing import cast
import datetime






T = TypeVar("T", bound="ComposerVersion")



@_attrs_define
class ComposerVersion:
    """ 
        Attributes:
            current (bool):
            label (str):
            name (str):
            saved_at (datetime.datetime):
            saved_by (str):
            size_bytes (int):
            source (ComposerVersionSource):
            version (int):
     """

    current: bool
    label: str
    name: str
    saved_at: datetime.datetime
    saved_by: str
    size_bytes: int
    source: ComposerVersionSource
    version: int





    def to_dict(self) -> dict[str, Any]:
        current = self.current

        label = self.label

        name = self.name

        saved_at = self.saved_at.isoformat()

        saved_by = self.saved_by

        size_bytes = self.size_bytes

        source = self.source.value

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "current": current,
            "label": label,
            "name": name,
            "savedAt": saved_at,
            "savedBy": saved_by,
            "sizeBytes": size_bytes,
            "source": source,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        current = d.pop("current")

        label = d.pop("label")

        name = d.pop("name")

        saved_at = datetime.datetime.fromisoformat(d.pop("savedAt"))




        saved_by = d.pop("savedBy")

        size_bytes = d.pop("sizeBytes")

        source = ComposerVersionSource(d.pop("source"))




        version = d.pop("version")

        composer_version = cls(
            current=current,
            label=label,
            name=name,
            saved_at=saved_at,
            saved_by=saved_by,
            size_bytes=size_bytes,
            source=source,
            version=version,
        )

        return composer_version

