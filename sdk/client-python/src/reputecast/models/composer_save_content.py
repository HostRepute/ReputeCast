from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.composer_save_content_source import ComposerSaveContentSource
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.composer_ab_test import ComposerABTest





T = TypeVar("T", bound="ComposerSaveContent")



@_attrs_define
class ComposerSaveContent:
    """ 
        Attributes:
            base_version (int):
            ab_test (ComposerABTest | Unset):
            cancel_approval (bool | Unset):
            force (bool | Unset):
            html (str | Unset):
            label (str | Unset):
            mjml (str | Unset):
            preheader (str | Unset):
            source (ComposerSaveContentSource | Unset):  Default: ComposerSaveContentSource.AUTOSAVE.
            stash (bool | Unset):
            subject (str | Unset):
     """

    base_version: int
    ab_test: ComposerABTest | Unset = UNSET
    cancel_approval: bool | Unset = UNSET
    force: bool | Unset = UNSET
    html: str | Unset = UNSET
    label: str | Unset = UNSET
    mjml: str | Unset = UNSET
    preheader: str | Unset = UNSET
    source: ComposerSaveContentSource | Unset = ComposerSaveContentSource.AUTOSAVE
    stash: bool | Unset = UNSET
    subject: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.composer_ab_test import ComposerABTest # noqa: PLC0415
        base_version = self.base_version

        ab_test: dict[str, Any] | Unset = UNSET
        if not isinstance(self.ab_test, Unset):
            ab_test = self.ab_test.to_dict()

        cancel_approval = self.cancel_approval

        force = self.force

        html = self.html

        label = self.label

        mjml = self.mjml

        preheader = self.preheader

        source: str | Unset = UNSET
        if not isinstance(self.source, Unset):
            source = self.source.value


        stash = self.stash

        subject = self.subject


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "baseVersion": base_version,
        })
        if ab_test is not UNSET:
            field_dict["abTest"] = ab_test
        if cancel_approval is not UNSET:
            field_dict["cancelApproval"] = cancel_approval
        if force is not UNSET:
            field_dict["force"] = force
        if html is not UNSET:
            field_dict["html"] = html
        if label is not UNSET:
            field_dict["label"] = label
        if mjml is not UNSET:
            field_dict["mjml"] = mjml
        if preheader is not UNSET:
            field_dict["preheader"] = preheader
        if source is not UNSET:
            field_dict["source"] = source
        if stash is not UNSET:
            field_dict["stash"] = stash
        if subject is not UNSET:
            field_dict["subject"] = subject

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.composer_ab_test import ComposerABTest # noqa: PLC0415
        d = dict(src_dict)
        base_version = d.pop("baseVersion")

        _ab_test = d.pop("abTest", UNSET)
        ab_test: ComposerABTest | Unset
        if isinstance(_ab_test,  Unset):
            ab_test = UNSET
        else:
            ab_test = ComposerABTest.from_dict(_ab_test)




        cancel_approval = d.pop("cancelApproval", UNSET)

        force = d.pop("force", UNSET)

        html = d.pop("html", UNSET)

        label = d.pop("label", UNSET)

        mjml = d.pop("mjml", UNSET)

        preheader = d.pop("preheader", UNSET)

        _source = d.pop("source", UNSET)
        source: ComposerSaveContentSource | Unset
        if isinstance(_source,  Unset):
            source = UNSET
        else:
            source = ComposerSaveContentSource(_source)




        stash = d.pop("stash", UNSET)

        subject = d.pop("subject", UNSET)

        composer_save_content = cls(
            base_version=base_version,
            ab_test=ab_test,
            cancel_approval=cancel_approval,
            force=force,
            html=html,
            label=label,
            mjml=mjml,
            preheader=preheader,
            source=source,
            stash=stash,
            subject=subject,
        )

        return composer_save_content

