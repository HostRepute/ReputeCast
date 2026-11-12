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
  from ..models.composer_merged import ComposerMerged





T = TypeVar("T", bound="ComposerSavedContent")



@_attrs_define
class ComposerSavedContent:
    """ 
        Attributes:
            content_hash (str):
            html (str):
            plain_text (str):
            saved_at (datetime.datetime):
            size_bytes (int):
            version (int):
            merged (ComposerMerged | Unset):
            recompiled (str | Unset):
            stashed_version (int | Unset):
            unchanged (bool | Unset):
     """

    content_hash: str
    html: str
    plain_text: str
    saved_at: datetime.datetime
    size_bytes: int
    version: int
    merged: ComposerMerged | Unset = UNSET
    recompiled: str | Unset = UNSET
    stashed_version: int | Unset = UNSET
    unchanged: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.composer_merged import ComposerMerged # noqa: PLC0415
        content_hash = self.content_hash

        html = self.html

        plain_text = self.plain_text

        saved_at = self.saved_at.isoformat()

        size_bytes = self.size_bytes

        version = self.version

        merged: dict[str, Any] | Unset = UNSET
        if not isinstance(self.merged, Unset):
            merged = self.merged.to_dict()

        recompiled = self.recompiled

        stashed_version = self.stashed_version

        unchanged = self.unchanged


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contentHash": content_hash,
            "html": html,
            "plainText": plain_text,
            "savedAt": saved_at,
            "sizeBytes": size_bytes,
            "version": version,
        })
        if merged is not UNSET:
            field_dict["merged"] = merged
        if recompiled is not UNSET:
            field_dict["recompiled"] = recompiled
        if stashed_version is not UNSET:
            field_dict["stashedVersion"] = stashed_version
        if unchanged is not UNSET:
            field_dict["unchanged"] = unchanged

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.composer_merged import ComposerMerged # noqa: PLC0415
        d = dict(src_dict)
        content_hash = d.pop("contentHash")

        html = d.pop("html")

        plain_text = d.pop("plainText")

        saved_at = datetime.datetime.fromisoformat(d.pop("savedAt"))




        size_bytes = d.pop("sizeBytes")

        version = d.pop("version")

        _merged = d.pop("merged", UNSET)
        merged: ComposerMerged | Unset
        if isinstance(_merged,  Unset):
            merged = UNSET
        else:
            merged = ComposerMerged.from_dict(_merged)




        recompiled = d.pop("recompiled", UNSET)

        stashed_version = d.pop("stashedVersion", UNSET)

        unchanged = d.pop("unchanged", UNSET)

        composer_saved_content = cls(
            content_hash=content_hash,
            html=html,
            plain_text=plain_text,
            saved_at=saved_at,
            size_bytes=size_bytes,
            version=version,
            merged=merged,
            recompiled=recompiled,
            stashed_version=stashed_version,
            unchanged=unchanged,
        )

        return composer_saved_content

