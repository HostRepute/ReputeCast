from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="ComposerReviewLink")



@_attrs_define
class ComposerReviewLink:
    """ 
        Attributes:
            created_at (datetime.datetime):
            created_by (str):
            expires_at (datetime.datetime):
            id (str):
            url (str): Shown to the workspace's members only
            version (int): The version reviewers see
     """

    created_at: datetime.datetime
    created_by: str
    expires_at: datetime.datetime
    id: str
    url: str
    version: int





    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        created_by = self.created_by

        expires_at = self.expires_at.isoformat()

        id = self.id

        url = self.url

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "createdAt": created_at,
            "createdBy": created_by,
            "expiresAt": expires_at,
            "id": id,
            "url": url,
            "version": version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        created_by = d.pop("createdBy")

        expires_at = datetime.datetime.fromisoformat(d.pop("expiresAt"))




        id = d.pop("id")

        url = d.pop("url")

        version = d.pop("version")

        composer_review_link = cls(
            created_at=created_at,
            created_by=created_by,
            expires_at=expires_at,
            id=id,
            url=url,
            version=version,
        )

        return composer_review_link

