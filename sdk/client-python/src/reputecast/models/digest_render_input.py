from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.digest_render_input_preview import DigestRenderInputPreview
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.digest_feed_block import DigestFeedBlock





T = TypeVar("T", bound="DigestRenderInput")



@_attrs_define
class DigestRenderInput:
    """ 
        Attributes:
            preview (DigestRenderInputPreview):
            block (DigestFeedBlock | Unset):
            mjml (str | Unset):
            preheader (str | Unset):
            sizes (bool | Unset):
            subject (str | Unset):
     """

    preview: DigestRenderInputPreview
    block: DigestFeedBlock | Unset = UNSET
    mjml: str | Unset = UNSET
    preheader: str | Unset = UNSET
    sizes: bool | Unset = UNSET
    subject: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.digest_feed_block import DigestFeedBlock # noqa: PLC0415
        preview = self.preview.value

        block: dict[str, Any] | Unset = UNSET
        if not isinstance(self.block, Unset):
            block = self.block.to_dict()

        mjml = self.mjml

        preheader = self.preheader

        sizes = self.sizes

        subject = self.subject


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "preview": preview,
        })
        if block is not UNSET:
            field_dict["block"] = block
        if mjml is not UNSET:
            field_dict["mjml"] = mjml
        if preheader is not UNSET:
            field_dict["preheader"] = preheader
        if sizes is not UNSET:
            field_dict["sizes"] = sizes
        if subject is not UNSET:
            field_dict["subject"] = subject

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.digest_feed_block import DigestFeedBlock # noqa: PLC0415
        d = dict(src_dict)
        preview = DigestRenderInputPreview(d.pop("preview"))




        _block = d.pop("block", UNSET)
        block: DigestFeedBlock | Unset
        if isinstance(_block,  Unset):
            block = UNSET
        else:
            block = DigestFeedBlock.from_dict(_block)




        mjml = d.pop("mjml", UNSET)

        preheader = d.pop("preheader", UNSET)

        sizes = d.pop("sizes", UNSET)

        subject = d.pop("subject", UNSET)

        digest_render_input = cls(
            preview=preview,
            block=block,
            mjml=mjml,
            preheader=preheader,
            sizes=sizes,
            subject=subject,
        )

        return digest_render_input

