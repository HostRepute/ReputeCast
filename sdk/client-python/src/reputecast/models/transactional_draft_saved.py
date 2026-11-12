from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="TransactionalDraftSaved")



@_attrs_define
class TransactionalDraftSaved:
    """ 
        Attributes:
            draft_changes (bool):
            draft_hash (str):
            draft_revision (int):
            html (str):
            plain_text (str):
            saved_at (datetime.datetime):
            size_bytes (int):
     """

    draft_changes: bool
    draft_hash: str
    draft_revision: int
    html: str
    plain_text: str
    saved_at: datetime.datetime
    size_bytes: int





    def to_dict(self) -> dict[str, Any]:
        draft_changes = self.draft_changes

        draft_hash = self.draft_hash

        draft_revision = self.draft_revision

        html = self.html

        plain_text = self.plain_text

        saved_at = self.saved_at.isoformat()

        size_bytes = self.size_bytes


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "draftChanges": draft_changes,
            "draftHash": draft_hash,
            "draftRevision": draft_revision,
            "html": html,
            "plainText": plain_text,
            "savedAt": saved_at,
            "sizeBytes": size_bytes,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        draft_changes = d.pop("draftChanges")

        draft_hash = d.pop("draftHash")

        draft_revision = d.pop("draftRevision")

        html = d.pop("html")

        plain_text = d.pop("plainText")

        saved_at = datetime.datetime.fromisoformat(d.pop("savedAt"))




        size_bytes = d.pop("sizeBytes")

        transactional_draft_saved = cls(
            draft_changes=draft_changes,
            draft_hash=draft_hash,
            draft_revision=draft_revision,
            html=html,
            plain_text=plain_text,
            saved_at=saved_at,
            size_bytes=size_bytes,
        )

        return transactional_draft_saved

